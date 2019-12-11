#!/bin/sh
source build/common.sh
set -x
docker rmi "$IMAGE_TAG" $IMAGE_TAG_LATEST
