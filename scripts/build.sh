#!/bin/bash

set -e

PARENT_PATH=$(dirname $(
	cd $(dirname $0)
	pwd -P
))

OS=$(eval "go env GOOS")
ARCH=$(eval "go env GOARCH")

pushd $PARENT_PATH
mkdir -p build
GO111MODULE=on go build -o build/silentiumd-$OS-$ARCH cmd/silentiumd/main.go
popd	
  