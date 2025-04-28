KUEUE_VERSION ?= v0.11.4
KIND_CLUSTER_NAME ?= kueue-ac-test
KUBE_VERSION ?= v1.31.6

start:
	kind create cluster --name ${KIND_CLUSTER_NAME} --image kindest/node:${KUBE_VERSION} --config ./kind/kind-config.yaml

install-kueue:
	kubectl --context kind-${KIND_CLUSTER_NAME} apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/download/${KUEUE_VERSION}/manifests.yaml

install:
	kubectl --context kind-${KIND_CLUSTER_NAME} apply -f ./k8s/queue.yaml

shutdown:
	kind delete cluster --name ${KIND_CLUSTER_NAME}

fetch-audit:
	docker exec ${KIND_CLUSTER_NAME}-control-plane cat /var/log/kubernetes/kube-apiserver-audit.log | jq -s '.[] | select(.verb == ("patch", "update", "create", "delete"))'

run-ac:
	cd ./admission-check/ && go run .

apply-wl:
	kubectl --context kind-${KIND_CLUSTER_NAME} apply -f ./k8s/wl.yaml

remove-wl:
	kubectl --context kind-${KIND_CLUSTER_NAME} rm -f ./k8s/wl.yaml
