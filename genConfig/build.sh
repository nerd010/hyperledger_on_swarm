#!/bin/bash
set -e 
wkdir=$(dirname -- $0)
cd $wkdir;
PWD=`pwd`
docker run --rm -v "$PWD":/usr/src/myapp -v "/Users/baoyangc/code/goprojs":/gopath -w /usr/src/myapp -e GOPATH=/gopath go1.9 go build
mkdir -p ../bin/linux-amd64
mv myapp ../bin/linux-amd64/genConfig
go build
mkdir -p ../bin/darwin-amd64
cp genConfig ../bin/darwin-amd64/genConfig