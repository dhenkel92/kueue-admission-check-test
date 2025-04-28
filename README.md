# Kueue Admission Check Test

While setting up custom admission checks, I noticed that when an admission
check state is set to retry, Kueue marks the workload as evicted but never
requeues it.

In theory, we would expect Kueue to release the quota, requeue the
workload, and restart the process from the beginning. However, it stops
somewhere along the way.

The new integration test aims to demonstrate the complete flow, but fails at
the state described above, where QuotaReserved=true and Evicted=true.

## How to replicate it

1. Start kind cluster

```
make start
```

2. Install kueue

```
make install-kueue
```

3. Wait for the `kueue-controller-manager` to be ready

```
kubectl --context kind-kueue-ac-test get pods -n kueue-system -w
```

4. Install custom kueue configurations

```
make install
```

5. Open a second shell and switch context to the kind cluster

```
kubectl config set-context kind-kueue-ac-test
```

6. Run admission check controller

```
make run-ac
```

7. Apply workload to cluster

```
make apply-wl
```

## Outcome

When describing the workload, you can see that the admission check was in a retry state and that Kueue reset the state.

```
Status:
  Admission:
    Cluster Queue:  retry-check-demo
    Pod Set Assignments:
      Count:  1
      Flavors:
        Cpu:     default-rf
        Memory:  default-rf
      Name:      main
      Resource Usage:
        Cpu:     1
        Memory:  128Mi
  Admission Checks:
    Last Transition Time:  2025-04-28T06:50:41Z
    Message:               Reset to Pending after eviction. Previously: Retry
    Name:                  retry-check
    State:                 Pending
  Conditions:
    Last Transition Time:  2025-04-28T06:50:41Z
    Message:               Quota reserved in ClusterQueue retry-check-demo
    Observed Generation:   1
    Reason:                QuotaReserved
    Status:                True
    Type:                  QuotaReserved
    Last Transition Time:  2025-04-28T06:50:41Z
    Message:               At least one admission check is false
    Observed Generation:   1
    Reason:                AdmissionCheck
    Status:                True
    Type:                  Evicted
```

Unfortunately, this is the last state it will reach.

With `make fetch-audit` you can get all the audit logs related to workloads and their status.
Here you can see which controller is making what changes to a workload and when. It's automatically prefiltered to only show mutating events.
