#!/bin/bash

sudo lsof -t -i:5000
cd ..
rm -rf main
go build main.go
cp main ./bin/master
cp main ./bin/node