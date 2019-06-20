#!/bin/bash
env GOOS=linux GOARCH=arm GOARM=6 go build -o read-values
scp read-values pi@energy.local:~
ssh pi@energy.local './read-values'
