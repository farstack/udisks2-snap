#!/bin/sh
set -ex

# Startup alongside udisksd can be racy so give it a head start
sleep 3
exec $SNAP/bin/ciborium
