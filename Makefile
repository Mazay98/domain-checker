# suppress output, run `make XXX V=` to be verbose
V := @

# Common
NAME = go.domain-checker
VCS = gitlab.lucky-team.pro
ORG = luckyads
VERSION := $(shell git describe --always --tags)
CURRENT_TIME := $(shell TZ="Europe/Moscow" date +"%d-%m-%y %T")

# Build
OUT_DIR = ./bin
MAIN_PKG = ./cmd/${NAME}
ACTION ?= build
GC_FLAGS = -gcflags 'all=-N -l'
LD_FLAGS = -ldflags "-s -v -w -X 'main.version=${VERSION}' -X 'main.buildTime=${CURRENT_TIME}'"
BUILD_CMD = CGO_ENABLED=1 go build -o ${OUT_DIR}/${NAME} ${LD_FLAGS} ${MAIN_PKG}
DEBUG_CMD = CGO_ENABLED=1 go build -o ${OUT_DIR}/${NAME} ${GC_FLAGS} ${MAIN_PKG}

# Docker
REGISTRY_URL = registry.lucky-team.pro
DOCKERFILE = deployments/docker/Dockerfile
DOCKER_IMAGE_NAME = ${REGISTRY_URL}/${ORG}/${NAME}
DOCKER_GOLANG_IMAGE = ${REGISTRY_URL}/luckyads/go.docker-images/alpine:1.18.0-v2

# Other
.DEFAULT_GOAL = build

.PHONY: build
build:
	@echo BUILDING PRODUCTION $(NAME)
	$(V)${BUILD_CMD}
	@echo DONE

.PHONY: build-debug
build-debug:
	@echo BUILDING DEBUG $(NAME)
	$(V)${DEBUG_CMD}
	@echo DONE

.PHONY: docker-build
docker-build:
	$(call run_in_docker,make ${ACTION})

.PHONY: lint
lint:
	$(V)golangci-lint run

.PHONY: docker-lint
docker-lint:
	$(call run_in_docker,make lint)

.PHONY: test
test: GO_TEST_FLAGS += -race
test:
	$(V)go test -mod=vendor $(GO_TEST_FLAGS) --tags=$(GO_TEST_TAGS) ./...

.PHONY: docker-test
docker-test:
	$(call run_in_docker, make test)

.PHONY: fulltest
fulltest: GO_TEST_TAGS += integration
fulltest: test

.PHONY: docker-fulltest
docker-fulltest:
	$(call run_in_docker, make fulltest)

.PHONY: generate
generate:
	$(V)go generate -x ./...

.PHONY: docker-generate
docker-generate:
	$(call run_in_docker,make generate)

.PHONY: clean
clean:
	@echo "Removing $(OUT_DIR)"
	$(V)rm -rf $(OUT_DIR)

.PHONY: vendor
vendor:
	$(V)GOPRIVATE=${VCS}/* go mod tidy -compat=1.17
	$(V)GOPRIVATE=${VCS}/* go mod vendor
	$(V)git add vendor go.mod go.sum

.PHONY: docker-build-local
docker-build-local:
	$(V)docker build -t ${DOCKER_IMAGE_NAME}:local -f ${DOCKERFILE} --build-arg ACTION=${ACTION} .

.PHONY: docker-build-push-image
docker-build-push-image:
	$(V)docker build -t ${DOCKER_IMAGE_NAME}:${VERSION} -f ${DOCKERFILE} --build-arg ACTION=${ACTION} .
	$(V)docker tag ${DOCKER_IMAGE_NAME}:${VERSION} ${DOCKER_IMAGE_NAME}:latest
	$(V)docker push ${DOCKER_IMAGE_NAME}:${VERSION}
	$(V)docker push ${DOCKER_IMAGE_NAME}:latest

CURR_REPO := /$(notdir $(PWD))
define run_in_docker
	$(V)docker run --rm \
		-v $(PWD):$(CURR_REPO) \
		-w $(CURR_REPO) \
		${DOCKER_GOLANG_IMAGE} $1
endef
