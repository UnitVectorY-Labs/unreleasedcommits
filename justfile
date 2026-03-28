
# Commands for unreleasedcommits
default:
  @just --list
# Build unreleasedcommits with Go
build:
  go build ./...

# Run tests for unreleasedcommits with Go
test:
  go clean -testcache
  go test ./...