package main

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	kueuev1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/workload"
)

const checkName = "retry-check"

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

type RetryAdmissionCheck struct {
	client client.Client
}

func (ctrl *RetryAdmissionCheck) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var wl kueuev1beta1.Workload
	if err := ctrl.client.Get(ctx, request.NamespacedName, &wl); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if !workload.HasQuotaReservation(&wl) || workload.IsFinished(&wl) || workload.IsEvicted(&wl) {
		return reconcile.Result{}, nil
	}

	logger := log.FromContext(ctx)
	logger.Info("handle workload admission check")

	check := workload.FindAdmissionCheck(wl.Status.AdmissionChecks, checkName)
	if check == nil || check.State != kueuev1beta1.CheckStatePending {
		return reconcile.Result{}, nil
	}

	wlPatch := workload.BaseSSAWorkload(&wl)
	wlPatch.ObjectMeta.ResourceVersion = wl.ObjectMeta.ResourceVersion
	workload.SetAdmissionCheckState(&wlPatch.Status.AdmissionChecks, kueuev1beta1.AdmissionCheckState{
		Name:               checkName,
		State:              kueuev1beta1.CheckStateRetry,
		LastTransitionTime: metav1.Now(),
		Message:            "test retry",
	}, clock.RealClock{})

	if err := ctrl.client.Status().Patch(ctx, wlPatch, client.Apply, client.FieldOwner("retry-test"), client.ForceOwnership); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

type ACReconciler struct {
	client client.Client
}

func (ctrl *ACReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	if request.Name != checkName {
		return reconcile.Result{}, nil
	}

	logger.Info("set admission check to active")

	var ac kueuev1beta1.AdmissionCheck
	if err := ctrl.client.Get(ctx, request.NamespacedName, &ac); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	currentCondition := apimeta.FindStatusCondition(ac.Status.Conditions, kueuev1beta1.AdmissionCheckActive)
	if currentCondition != nil && currentCondition.Status == metav1.ConditionTrue {
		return reconcile.Result{}, nil
	}

	if currentCondition != nil {
		currentCondition.Status = metav1.ConditionTrue
	} else {
		ac.Status.Conditions = append(ac.Status.Conditions, metav1.Condition{
			Type:               kueuev1beta1.AdmissionCheckActive,
			LastTransitionTime: metav1.Now(),
			Status:             metav1.ConditionTrue,
			Reason:             "Active",
			Message:            "The admission check is active",
			ObservedGeneration: ac.Generation,
		})
	}

	if err := ctrl.client.Status().Update(ctx, &ac); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func main() {
	log.SetLogger(zap.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	handleError(err)

	err = kueuev1beta1.AddToScheme(mgr.GetScheme())
	handleError(err)

	reconciler := &RetryAdmissionCheck{
		client: mgr.GetClient(),
	}
	_, err = ctrl.NewControllerManagedBy(mgr).
		For(&kueuev1beta1.Workload{}).
		Named("WorkloadACController").
		Build(reconciler)
	handleError(err)

	acReconciler := &ACReconciler{
		client: mgr.GetClient(),
	}
	_, err = ctrl.NewControllerManagedBy(mgr).
		For(&kueuev1beta1.AdmissionCheck{}).
		Named("ACController").
		Build(acReconciler)
	handleError(err)

	err = mgr.Start(ctrl.SetupSignalHandler())
	handleError(err)
}
