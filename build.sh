#!/bin/bash

cd `dirname $0`

go build -ldflags "-s -w"