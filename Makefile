# Get the current version
VERSION=$(shell cat ./VERSION)
WORKDIR=$(shell pwd)
UID=$(shell echo $uid)

.PHONY: help

# Output help for every task
help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
.DEFAULT_GOAL := help

build: ## Build the container image (with cache)
	buildah bud --layers --build-arg VERSION="$(VERSION)" \
		--tag=git.rpjosh.de/ncDocConverter:v$(VERSION)-dev \
		-f Dockerfile .
build-nc: ## Build the container image (without cache)
	buildah bud --layers --no-cache --build-arg VERSION="$(VERSION)" \
		--tag=git.rpjosh.de/ncDocConverter:v$(VERSION)-dev \
		-f Dockerfile .

run:  ## Run the container with
	@ make stop > /dev/null 2>&1 || true
	@ podman run -it --name ncDocConverter --userns=keep-id --cap-drop ALL -p 40001:40001 -e PORT=40001 \
		-e DATA_FILE='./config/data.json' \
		-v "$(WORKDIR)/ncConverter.json:/config/data.json" \
		-v "$(WORKDIR)/config.yaml:/config/config.yaml" \
		git.rpjosh.de/ncDocConverter:v$(VERSION)-dev
stop: ## Stop and removes a previous started container
	@ podman stop ncDocConverter; podman rm ncDocConverter

clear-images: ## Remove all previously build images and all intermediate images created by this makefile
	podman rmi $$(podman images -a | grep -e '<none>' -e '\/ncdocconverter-.*' | awk '{ print $3 }') -f


# Required secrets:
#  Android Key Store (jks) file          	-	androidKeystore
#  Android Key Store password (cleartext)   -	androidKeystorePassword