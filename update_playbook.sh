#!/bin/sh
SERVICE=healthcheck-watcher:watcher

docker run --rm \
-v "$(pwd)/playbook:/playbook" \
-w /playbook \
"$STRATSYS_CR_URL/stratsys-envhandler:latest" \
--destination dev \
--localSource /playbook \
--imageTag "$IMAGE_NAME" \
--token "$GITHUB_PLAYBOOK_TOKEN" \
--url "$GITHUB_PLAYBOOK_URL" \
--email "$GITHUB_PLAYBOOK_EMAIL" \
--author "$GITHUB_PLAYBOOK_AUTHOR" \
--verbosity 4 \
copy "$SERVICE"
