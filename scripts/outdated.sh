#!/usr/bin/env sh

SCRIPT_PATH=$(realpath "$0")
SCRIPTS_DIR_PATH=$(dirname "${SCRIPT_PATH}")
LIB_DIR_PATH="${SCRIPTS_DIR_PATH}/lib"

. "${LIB_DIR_PATH}/logging.sh"

main() {
  output=$(
    go mod edit -json | jq -r '.Require[].Path' | \
    xargs go list -u -m -f '{{if .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}'
  )

  if [ -z "${output}" ]; then
    log_info "All dependencies are up-to-date"
  else
    echo "${output}"
  fi
}

main "$@"
