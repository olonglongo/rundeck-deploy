GO    := GO15VENDOREXPERIMENT=1 go
pkgs   = $(shell $(GO) list ./... | grep -v /vendor/)

BASE_DIR				?= $(shell pwd)
BIN_NAME				?= $(shell basename $(BASE_DIR))
GOPATH					?= $(BASE_DIR)
BIN_DIR                 ?= $(GOPATH)/bin
UPX_BIN					?= $(shell which upx)
DOCKER_BUILD_IMAGE		?= golang:crossbuild
DOCKER_IMAGE_NAME       ?= $(shell pwd)
DOCKER_IMAGE_TAG        ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))
TAG 					:= $(shell echo `if [ "$(TRAVIS_BRANCH)" = "master" ] || [ "$(TRAVIS_BRANCH)" = "" ] ; then echo "latest"; else echo $(TRAVIS_BRANCH) ; fi`)

all: format asset crossbuild

init:
	@rm -rf go.mod go.sum vendor
	@go mod init
	@go get k8s.io/client-go@v0.18.0
	@$(GO) get -u github.com/a-urth/go-bindata/...
	@$(shell rm -rf asset/asset.go)
	@$(shell go-bindata -o=asset/asset.go -pkg=asset conf/...)
	@go mod vendor

test:
	@echo ">> running tests"
	@$(GO) test -short $(pkgs)

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

style:
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

asset:
	@echo ">> caching asset file"
	@$(GO) get -u github.com/a-urth/go-bindata/...
	@$(shell rm -rf asset/asset.go)
	@$(shell go-bindata -o=asset/asset.go -pkg=asset conf/...)

build: 
	@echo ">> building binaries"
	@$(GO) build -v -o $(BIN_DIR)/$(BIN_NAME)

release: 
	@echo ">> building binaries"
	@$(GO) build -ldflags="-s -w" -v -o $(BIN_DIR)/$(BIN_NAME)
	@echo ">> compress binaries"
	@$(UPX_BIN) -9 -v $(BIN_DIR)/$(BIN_NAME) && $(UPX_BIN) -t $(BIN_DIR)/$(BIN_NAME)

crossbuild:
	@echo ">> crossbuilding binaries"
	@docker run --rm -it -v $(BASE_DIR):/build -w /build $(DOCKER_BUILD_IMAGE) \
		go build -mod=mod -v -ldflags="-s -w" -o bin/$(BIN_NAME)
	@docker run --rm -it -v $(BASE_DIR):/build -w /build $(DOCKER_BUILD_IMAGE) \
		upx -9 bin/$(BIN_NAME)
	@scp bin/$(BIN_NAME) ubuntu@10.228.3.40:~/deploy
	@scp bin/$(BIN_NAME) ubuntu@10.228.3.31:~/deploy

docker:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

push:
	@echo ">> pushing docker image, $(DOCKER_USERNAME),$(DOCKER_IMAGE_NAME),$(TAG)"
	@docker login -u $(DOCKER_USERNAME) -p $(DOCKER_PASSWORD)
	@docker tag "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" "$(DOCKER_USERNAME)/$(DOCKER_IMAGE_NAME):$(TAG)"
	@docker push "$(DOCKER_USERNAME)/$(DOCKER_IMAGE_NAME):$(TAG)"

github-release:
	@GOOS=$(shell uname -s | tr A-Z a-z) \
		GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m))) \
		$(GO) get -u github.com/aktau/github-release

.PHONY: all style format build test vet docker crossbuild release asset