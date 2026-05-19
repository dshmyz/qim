#!/bin/sh

set -e

SUDOERS_FILE="/etc/sudoers.d/qim-update"

if [ -f "$SUDOERS_FILE" ]; then
  rm -f "$SUDOERS_FILE"
  echo "QIM: sudoers rule removed from $SUDOERS_FILE"
fi
