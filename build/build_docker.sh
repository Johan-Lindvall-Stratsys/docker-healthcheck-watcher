#!/bin/bash
source build/common.sh
set -x
LATEST_TAG_ARG=""
[[ ! -z $"IMAGE_TAG_LATEST" ]] && LATEST_TAG_ARG="-t "$IMAGE_TAG_LATEST""
docker build --rm -t "$IMAGE_TAG" $LATEST_TAG_ARG $DOCKER_BUILD_ARGS .
