
# Commands for lockboxkms
default:
  @just --list
# Build lockboxkms with Go
build:
  go build ./...

# Run tests for lockboxkms with Go
test:
  go clean -testcache
  go test ./...