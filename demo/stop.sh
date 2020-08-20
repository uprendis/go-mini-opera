#!/bin/bash

PROG=miniopera

# kill all miniopera processes
killall "${PROG}"
sleep 3

# remove demo data
rm -rf /tmp/miniopera-demo/datadir/
