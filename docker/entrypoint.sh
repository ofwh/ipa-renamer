#!/bin/sh
set -eu

set -- /app/ipa-renamer -i /app/in -o /app/out "$@"

case "${WATCH:-}" in
  1 | true | yes | on)
    # -t only applies to watch mode.
    if [ -n "${IDLE_TIMEOUT:-}" ]; then
      set -- "$@" -t "$IDLE_TIMEOUT"
    fi
    set -- "$@" -w
    ;;
esac

exec "$@"
