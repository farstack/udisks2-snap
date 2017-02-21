#!/bin/sh -ex

SUDO=
if [ $(id -u) -ne 0 ]; then
	SUDO=sudo
fi

# Install all necessary build dependencies in our bare build environment
$SUDO apt update
$SUDO apt install -y --force-yes \
	golang-go \
	bzr

export GOPATH=`mktemp -d`

mkdir -p $GOPATH/src/launchpad.net
ln -sf $PWD $GOPATH/src/launchpad.net/ciborium

go get -v launchpad.net/ciborium/cmd/ciborium
go get -v -t launchpad.net/ciborium/cmd/ciborium

go build -v launchpad.net/ciborium/cmd/ciborium

go test -v launchpad.net/ciborium/cmd/ciborium
go test -v launchpad.net/ciborium/udisks2
