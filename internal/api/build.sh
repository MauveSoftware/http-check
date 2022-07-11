#!/bin/bash
protoc --go_out=paths=source_relative,plugins=grpc:. service.proto
