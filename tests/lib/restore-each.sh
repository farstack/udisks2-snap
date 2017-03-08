#!/bin/bash

. $TESTSLIB/snap-names.sh
. $TESTSLIB/utilities.sh

# Remove all snaps not being the core, gadget, kernel or snap we're testing
for snap in /snap/*; do
	snap="${snap:6}"
	case "$snap" in
		"bin" | "$gadget_name" | "$kernel_name" | "$core_name" | "$SNAP_NAME" )
			;;
		*)
			snap remove "$snap"
			;;
	esac
done

# Cleanup all configuration files from the snap so that we have
# a fresh start for the next test
rm -rf /var/snap/$SNAP_NAME/common/
mkdir -p /var/snap/$SNAP_NAME/common/
real_current=$(readlink -f /var/snap/$SNAP_NAME/current/)
rm -rf $real_current
mkdir -p $real_current

# Ensure we have the same state for snapd as we had before
systemctl stop snapd.service snapd.socket
rm -rf /var/lib/snapd/*
tar xzf $SPREAD_PATH/snapd-state.tar.gz -C /
rm -rf /root/.snap
systemctl start snapd.service snapd.socket

# For debugging dump all snaps and connected slots/plugs
snap list
snap interfaces
