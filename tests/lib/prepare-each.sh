#!/bin/sh
. $TESTSLIB/utilities.sh
snap_install udisks2
snap connect udisks2:client udisks2:service
