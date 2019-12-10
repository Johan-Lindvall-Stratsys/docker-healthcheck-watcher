#!/bin/sh
docker run --rm \
-v ./stack:/playbook/stack \
-w /playbook \
"$STRATSYS_CR_URL/stratsys-envhandler:latest" \ 
--branch dev \
--destination dev \
--localSource /playbook \
--token "$GITHUB_PLAYBOOK_TOKEN" \
--url "$GITHUB_PLAYBOOK_URL" \
--email "$GITHUB_PLAYBOOK_EMAIL" \
--author "$GITHUB_PLAYBOOK_AUTHOR" \
--verbosity 4 \
copy healthcheck-watcher:watcher
