#!/bin/sh

./web/app/node_modules/.bin/nodemon --delay 10s -e go,html --ignore web/app/ --signal SIGTERM --exec go run ./cmd/ncDocConverth || exit 1