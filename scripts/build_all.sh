#!/bin/bash

set -e

PARENT_PATH=$(dirname $(
    cd $(dirname $0)
    pwd -P
))

declare -a OS=("darwin" "linux")
declare -a ARCH=("amd64" "arm64")

pushd $PARENT_PATH
mkdir -p build

for os in "${OS[@]}"; do
  for arch in "${ARCH[@]}"; do
    echo "Building for $os $arch"
    GOOS=$os GOARCH=$arch go build -o build/silentiumd-$os-$arch cmd/silentiumd/main.go
  done
done

popd