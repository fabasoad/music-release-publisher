#!/usr/bin/env sh

main() {
  tmp_file=$(mktemp)
  grep -v "/mock.go" coverage.out > tmp_file
  mv tmp_file coverage.out
  go tool cover -$1=coverage.out
}

main "$@"
