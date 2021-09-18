#!/bin/bash

env GOOS=linux GOARCH=amd64 go build ../main.go
mv main ../bin/cb-binary