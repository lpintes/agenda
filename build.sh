#!/bin/sh
# Builds agenda.exe (GUI) and agenda-check.exe (console check).
# Usage: ./build.sh [-test]   -test also runs go vet and the unit tests.
set -e
cd "$(dirname "$0")"

if [ "$1" = "-test" ]; then
	go vet ./...
	go test ./...
fi

go build -ldflags="-H windowsgui" -o agenda.exe .
go build -o agenda-check.exe ./cmd/agenda-check
echo "Built agenda.exe and agenda-check.exe"
