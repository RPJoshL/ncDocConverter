#!/bin/sh

GREEN='\033[0;32m'
NC='\033[0m'

nodemon --delay 1s -e go,html,yaml --ignore web/app/ --signal SIGTERM --quiet --exec \
'echo "\n'"$GREEN"'[Restarting]'"$NC"'" && go run -ldflags "-X main.version="$(cat VERSION)"" ./cmd/ncDocConverth' -- "$@" "|| exit 1"