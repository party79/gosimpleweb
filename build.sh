#!/bin/bash

MAJVER=1
if [ "X$1" != "X" ]; then
    MAJVER=$1
fi

MINVER=0
if [ "X$2" != "X" ]; then
    MINVER=$2
fi

BUILDDATE=`date +%Y-%m-%d\ %H:%M`

go get github.com/mitchellh/gox
gox -os="darwin linux windows" -arch="386 amd64" -ldflags "-X main.majversion=$MAJVER -X main.minversion=$MINVER -X \"main.builddate=$BUILDDATE\"" -output="build/${MAJVER}.${MINVER}/gosimpleweb-${MAJVER}.${MINVER}-{{.OS}}-{{.Arch}}"
