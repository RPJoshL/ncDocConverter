#!/bin/bash

## This script deploys the build files to the given directory.
## 
## You can define all below variables also via environment variables
## with the prefix NC_DOC_CONVERTER_
## Example: NC_DOC_CONVERTER_ROOT_DIRECTORY="/home/ncDocConverter/"
## You can set theses for example in jenkins.

# Directory in which the executable file should be installed
ROOT_DIRECTORY="/usr/share/RPJosh/ncDocConverter/"

# Configuration directory containing the configuration file
CONFIGURATION_DIRECTORY="/etc/ncDocConverter/"

# User to execute the program
USER=ncDocConverter

# The service name for systemctl
SERVICE_NAME=ncDocConverter

# Arch and operating system
ARCH="amd64"

version="$(cat VERSION)"

overwriteVars() {
    vars=( ROOT_DIRECTORY CONFIGURATION_DIRECTORY USER SERVICE_NAME )
    for var in "${vars[@]}"; do
        envVar="NC_DOC_CONVERTER_"$var""
        #envVar="$(eval "echo \$$envVar")"
        envVar="${!envVar}"
	    if [ -n "$envVar" ]; then
            declare -g $var="$envVar"
        fi
    done
}

# Overwrite environment variables
overwriteVars
set -e

## Stop service
systemctl is-active --quiet "$SERVICE_NAME" && systemctl stop "$SERVICE_NAME"

## Copy binary
mkdir -p "$ROOT_DIRECTORY"
cp "ncDocConverth-"$version"-"$ARCH"" ""$ROOT_DIRECTORY"ncDocConverth"
chown -R ""$USER":"$USER"" "$ROOT_DIRECTORY"

## Copy configuration
mkdir -p "$CONFIGURATION_DIRECTORY"
if ! [ -e ""$CONFIGURATION_DIRECTORY"config.yaml" ]; then
    cp ./configs/config.yaml ""$CONFIGURATION_DIRECTORY"config.yaml.example"
    cp ./configs/ncConverter.hjson ""$CONFIGURATION_DIRECTORY"config.hjson.example" 
fi
chown -R ""$USER":"$USER"" "$CONFIGURATION_DIRECTORY"
chmod -R 0600 "$CONFIGURATION_DIRECTORY"
chmod 0700 "$CONFIGURATION_DIRECTORY"

## Start service
systemctl start "$SERVICE_NAME"

exit 0