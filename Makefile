REGISTRY ?= my-registry
VERSION ?= v1.0.1

.PHONY: build-cli build-agent build-server build docker-build docker-build-all

build: build-cli build-agent build-server

build-cli:
	go build -o bin/mydeploy ./cmd/cli

build-agent:
	go build -o bin/mydeploy-agent ./cmd/agent

# Docker build targets
docker-build-auth:
	docker build -t $(REGISTRY)/auth-service:$(VERSION) -f cmd/auth-service/Dockerfile .

docker-build-agent-svc:
	docker build -t $(REGISTRY)/agent-service:$(VERSION) -f cmd/agent-service/Dockerfile .

docker-build-deploy:
	docker build -t $(REGISTRY)/deploy-service:$(VERSION) -f cmd/deploy-service/Dockerfile .

docker-build-template:
	docker build -t $(REGISTRY)/template-service:$(VERSION) -f cmd/template-service/Dockerfile .

docker-build-gateway:
	docker build -t $(REGISTRY)/gateway-service:$(VERSION) -f cmd/gateway/Dockerfile .

docker-build-all: docker-build-auth docker-build-agent-svc docker-build-deploy docker-build-template docker-build-gateway

# Helm & K8s targets
HELM_RELEASE_NAME ?= my-release
CHART_PATH ?= ./charts/my-deploy

k8s-db-setup:
	kubectl apply -f $(CHART_PATH)/external-db.yaml

k8s-install: k8s-db-setup
	helm install $(HELM_RELEASE_NAME) $(CHART_PATH)

k8s-uninstall:
	helm uninstall $(HELM_RELEASE_NAME)
	kubectl delete svc postgres-auth postgres-agent postgres-deploy || true
	kubectl delete endpoints postgres-auth postgres-agent postgres-deploy || true

k8s-reinstall: k8s-uninstall k8s-install

k8s-status:
	kubectl get pods
	kubectl get svc
	kubectl get ingress

k8s-logs-auth:
	kubectl logs -l app=auth

k8s-logs-gateway:
	kubectl logs -l app=gateway

k8s-restart:
	kubectl rollout restart deployment
