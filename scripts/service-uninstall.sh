#!/bin/bash

test $# = 1 || { echo "Need 1 parameter: service file to install"; exit 1; }
FULL_NAME=$(readlink -f "$1")
FILE_NAME=$(basename "$FULL_NAME")

rm "/etc/systemd/system/multi-user.target.wants/$FILE_NAME"
rm "/etc/systemd/system/$FILE_NAME"
