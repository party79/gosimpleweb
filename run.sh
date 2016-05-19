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

go build -ldflags "-X main.majversion=$MAJVER -X main.minversion=$MINVER-beta -X \"main.builddate=$BUILDDATE\"" -o "gosimpleweb"

./gosimpleweb
