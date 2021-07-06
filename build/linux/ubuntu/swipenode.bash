#!/bin/bash

apt-get update
apt-get -y upgrade
apt install -y curl vim git build-essential jq

wget https://dl.google.com/go/go1.15.5.linux-amd64.tar.gz
tar -xvf go1.15.5.linux-amd64.tar.gz
mv go /usr/local
rm go1.15.5.linux-amd64.tar.gz # cleanup

export GOROOT=/usr/local/go
export GOPATH=~/go
export GOBIN=$GOPATH/bin
export PATH=$GOBIN:$GOROOT/bin:$PATH

git clone https://github.com/SwipeWallet/swipechain-node.git ~/go/src/github.com/SwipeWallet/swipechain-node

cd ~/go/src/github.com/SwipeWallet/swipechain-node
go get -v
make install tools

cp $GOBIN/swiped /usr/bin/
cp $GOBIN/swipecli /usr/bin/
cp $GOBIN/observed /usr/bin/
cp $GOBIN/signd /usr/bin/
cp $GOBIN/generate /usr/bin/
