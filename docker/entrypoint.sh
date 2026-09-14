#!/bin/sh
set -eu

set -- /app/ipa-renamer -i /app/in -o /app/out

case "${WATCH:-1}" in
  0) ;;
  *) set -- "$@" -t "${IDLE_TIMEOUT:-5}" -w ;;
esac

case "${RECURSIVE:-1}" in
  0) ;;
  *) set -- "$@" -r ;;
esac

case "${VERBOSE:-0}" in
  1) set -- "$@" -v ;;
esac

exec "$@"
