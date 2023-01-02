#!/bin/sh

nodemon --delay 1s -e go,html,yaml --ignore web/app/ --signal SIGTERM --exec 'go run ./cmd/ncDocConverth || exit 1'