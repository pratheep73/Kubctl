.ONESHELL:
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell GO111MODULE=off go run ${COMMONDIR}/time.go)
VERSION := $(or ${GITHUB_TAG_NAME},$(shell git describe --tags --exact-match 2> /dev/null || git symbolic-ref -q --short HEAD || git rev-parse --short HEAD))

# Image URL to use all building/pushing image targets
DOCKER_TAG := $(or ${GITHUB_TAG_NAME}, latest)
DOCKER_IMG ?= ghcr.io/metal-stack/firewall-controller:${DOCKER_TAG}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
# this version is used to include template from the metal-networker to the firewall-controller
# version should be not that far away from the compile dependency in go.mod
METAL_NETWORKER_VERSION := v0.6.4

# Kubebuilder installation environment variables
KUBEBUILDER_DOWNLOAD_URL := https://github.com/kubernetes-sigs/kubebuilder/releases/download
KUBEBUILDER_VER := 2.3.1
KUBEBUILDER_ASSETS := /usr/local/bin

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: firewall-controller

# Run tests
test: generate fmt vet manifests
	go test ./... -short -coverprofile cover.out

test-all: generate fmt vet manifests
	go test ./... -v -coverprofile cover.out

test-integration: generate fmt vet manifests
	go test ./... -v Integration

clean:
	rm -rf bin/* pkg/network/frr.firewall.tpl

# Build firewall-controller binary
firewall-controller: generate fmt vet test
	CGO_ENABLED=0 go build \
		-tags netgo \
		-trimpath \
		-ldflags \
			"-X 'github.com/metal-stack/v.Version=$(VERSION)' \
			-X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
			-X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
			-X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'" \
		-o bin/firewall-controller main.go
	strip bin/firewall-controller
	sha256sum bin/firewall-controller > bin/firewall-controller.sha256

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${DOCKER_IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen fetch-template
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Fetch firewall template
fetch-template:
	wget https://raw.githubusercontent.com/metal-stack/metal-networker/${METAL_NETWORKER_VERSION}/pkg/netconf/tpl/frr.firewall.tpl -O ./pkg/network/frr.firewall.tpl

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen manifests
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build:
	docker build . -t ${DOCKER_IMG}

# Push the docker image
docker-push:
	docker push ${DOCKER_IMG}

kubebuilder:
	set -ex \
 		&& mkdir -p /tmp/kubebuilder /usr/local/bin \
 		&& curl -L ${KUBEBUILDER_DOWNLOAD_URL}/v${KUBEBUILDER_VER}/kubebuilder_${KUBEBUILDER_VER}_linux_amd64.tar.gz -o /tmp/kubebuilder-${KUBEBUILDER_VER}-linux-amd64.tar.gz \
 		&& tar xzvf /tmp/kubebuilder-${KUBEBUILDER_VER}-linux-amd64.tar.gz -C /tmp/kubebuilder --strip-components=1 \
 		&& mv /tmp/kubebuilder/bin/* ${KUBEBUILDER_ASSETS}/

# find or download controller-gen
# download controller-gen if necessary
.PHONY: controller-gen
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
