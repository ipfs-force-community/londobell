#!/bin/sh
set -e

make gen-index
make gen-model
make gen-diff

echo "check for file changes"
commit=$(git describe --always --match=NeVeRmAtCh --dirty)
if [[ $commit == *"-dirty" ]]; then
	echo "un-committed changes"
	exit 1
fi
