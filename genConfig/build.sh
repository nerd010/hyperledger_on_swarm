#!/bin/bash
PWD=`pwd`
docker run --rm -v "$PWD":/usr/src/myapp -v "/Users/baoyangc/code/goprojs":/gopath -w /usr/src/myapp -e GOPATH=/gopath go1.9 go build
mv myapp genConfig