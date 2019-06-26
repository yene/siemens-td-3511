#!/bin/bash
set -e
set -x
ARG1=${1:-"energy.local"}
# compile for Raspberry Pi 3
env GOOS=linux GOARCH=arm GOARM=6 go build -o read-values
scp read-values "pi@${ARG1}":~
ssh "pi@${ARG1}" "~/read-values"
