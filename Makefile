# use bash as shell within makefile
SHELL := /bin/bash

# go variables
MODFLAGS := -mod=readonly

# default to using dev docker files
COMPOSE := docker-compose -f docker-compose.yml -f docker-compose.dev.yml
COMPOSE_FLAGS := --remove-orphans

# set server name to localhost. This is used in the nginx config to route data correctly
SERVER_NAME ?= localhost

deploy:
	bash scripts/deploy.sh

rebuild-txnotify: lint
	${COMPOSE} up ${COMPOSE_FLAGS} --detach --build txnotify

prepare:
	bash proto/gen_protos.sh
	cd frontend && yarn install && yarn gen-typescript-api && yarn gen-docs

lint:
	golangci-lint run
	cd frontend && yarn tsc

start-frontend:
	cd frontend && yarn start-background

dev-serve: prepare start-frontend
	${COMPOSE} up ${COMPOSE_FLAGS} --detach --build \
		certbot nginx bitcoind postgres txnotify docs

