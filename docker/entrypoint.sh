#!/bin/sh
set -eu

# Reads the IPA_RENAMER_* variables and starts the binary with the matching
# CLI arguments, so containers configure the tool through environment variables
# (.env / environment) rather than a fixed command line.
#
#   IPA_RENAMER_WATCH=1   adds -w                (default 1)
#   IPA_RENAMER_INPUT=    adds -i <dir>
#   IPA_RENAMER_OUTPUT=   adds -o <dir>
#   IPA_RENAMER_TIME=     adds -t <seconds>
set -- /usr/local/bin/ipa-renamer

[ -n "${IPA_RENAMER_INPUT:-}" ] && set -- "$@" -i "$IPA_RENAMER_INPUT"
[ -n "${IPA_RENAMER_OUTPUT:-}" ] && set -- "$@" -o "$IPA_RENAMER_OUTPUT"
[ -n "${IPA_RENAMER_TIME:-}" ] && set -- "$@" -t "$IPA_RENAMER_TIME"

case "${IPA_RENAMER_WATCH:-1}" in
  0 | false | no | off) ;;
  *) set -- "$@" -w ;;
esac

exec "$@"
