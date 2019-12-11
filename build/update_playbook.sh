#!/bin/bash
source build/common.sh

[[ -z "$PLAYBOOK_SERVICE" ]] && exit 0

( [[ -z "$GITHUB_PLAYBOOK_TOKEN" ]] || [[ -z "$GITHUB_PLAYBOOK_URL" ]] || [[ -z "$GITHUB_PLAYBOOK_EMAIL" ]] || [[ -z "$GITHUB_PLAYBOOK_AUTHOR" ]] ) &&
   echo "GITHUB_PLAYBOOK_TOKEN, GITHUB_PLAYBOOK_URL, GITHUB_PLAYBOOK_EMAIL and GITHUB_PLAYBOOK_AUTHOR must be set." &&
   exit 1

docker run --rm \
-v "$(pwd)/playbook:/playbook" \
-w /playbook \
"$CONTAINER_REGISTRY/stratsys-envhandler:latest" \
--destination dev \
--localSource /playbook \
--imageTag "$IMAGE_TAG" \
--token "$GITHUB_PLAYBOOK_TOKEN" \
--url "$GITHUB_PLAYBOOK_URL" \
--email "$GITHUB_PLAYBOOK_EMAIL" \
--author "$GITHUB_PLAYBOOK_AUTHOR" \
--verbosity 4 \
copy "$PLAYBOOK_SERVICE"
