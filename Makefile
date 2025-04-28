start:
	kind create cluster --name kueue-ac-test --image kindest/node:v1.31.6@sha256:28b7cbb993dfe093c76641a0c95807637213c9109b761f1d422c2400e22b8e87 --config ./kind/kind-config.yaml

install-kueue:
	kubectl --context kind-kueue-ac-test apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/download/v0.11.4/manifests.yaml

install:
	kubectl --context kind-kueue-ac-test apply -f ./k8s/queue.yaml

shutdown:
	kind delete cluster --name kueue-ac-test

fetch-audit:
	docker exec kueue-ac-test-control-plane cat /var/log/kubernetes/kube-apiserver-audit.log | jq -s '.'

run-ac:
	cd ./admission-check/ && go run .
