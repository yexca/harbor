.DEFAULT_GOAL := help

GO ?= go
NODE ?= node
DOCKER ?= docker
GO_PARALLEL ?= 2
DOCKER_IMAGE ?= harbor:check
COMPOSE_PROJECT ?= harbor
COMPOSE = $(DOCKER) compose -p $(COMPOSE_PROJECT)

ifeq ($(OS),Windows_NT)
EXE := .exe
endif

.PHONY: help run build format format-check test test-race vet backend-check web-check docs-check scripts-test sensitive-check check ci-local ci docker-build docker-smoke docker-up docker-down docker-status docker-logs

help:
	@$(NODE) scripts/check.mjs help

run:
	$(GO) run ./server

build:
	$(GO) build -p $(GO_PARALLEL) -trimpath -ldflags="-s -w" -o bin/harbor$(EXE) ./server

format:
	$(NODE) scripts/check.mjs format

format-check:
	$(NODE) scripts/check.mjs format-check

test:
	$(GO) test -p $(GO_PARALLEL) ./...

test-race:
	$(GO) test -p $(GO_PARALLEL) -race ./...

vet:
	$(GO) vet -p $(GO_PARALLEL) ./...

backend-check: format-check vet test

web-check:
	$(NODE) --check server/web/app.js

docs-check:
	$(NODE) scripts/check.mjs docs

scripts-test:
	$(NODE) --test scripts/check.test.mjs scripts/release.test.mjs

sensitive-check:
	$(NODE) scripts/check.mjs sensitive

check: backend-check web-check docs-check scripts-test

ci-local: check build

ci: ci-local test-race docker-smoke

docker-build:
	$(DOCKER) build -t $(DOCKER_IMAGE) .

docker-smoke: docker-build
	$(NODE) scripts/smoke.mjs $(DOCKER_IMAGE) $(DOCKER)

docker-up:
	$(COMPOSE) up -d --build

docker-down:
	$(COMPOSE) down

docker-status:
	$(COMPOSE) ps

docker-logs:
	$(COMPOSE) logs --tail=100
