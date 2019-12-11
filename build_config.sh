#!/bin/sh
CONTAINER_REGISTRY=stratsys.azurecr.io
CONTAINER_REGISTRY_REPOSITORY=docker-healthcheck-watcher
PLAYBOOK_SERVICE=healthcheck-watcher:watcher
PUSH_LATEST=0
#DOCKER_BUILD_ARGS="--build-arg FOO=BAR"