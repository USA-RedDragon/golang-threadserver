#!/bin/bash

protoc -I. --go_out=. protobuf/*.proto
go build
