#!/bin/sh

nodemon --delay 1s -e go,html --ignore web/app/ --exec go run ./cmd/ncDocConverth --signal SIGTERM