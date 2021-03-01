#!/bin/bash

set -e

if [[ -f "./build/genesis.go" ]]
then
	echo "build dir exists";
	exit 0
fi

echo "make link for build dir"
ln -s `go run tool/import.go` ./build
