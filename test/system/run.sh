#!/bin/bash -eux
# supposed to be run from the root of the project

go build -o build/out/mposter cmd/mposter/main.go 
# -count=1 to disable caching since messed up ocassionally: run the system test against old build before producing a new binary.
go test -count=1 ./test/system/