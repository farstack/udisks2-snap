#!/bin/sh
snap_install() {
	name=$1
	if [ -n "$SNAP_CHANNEL" ] ; then
		# Don't reinstall if we have it installed already
		if ! snap list | grep $name ; then
			snap install --$SNAP_CHANNEL $name
		fi
	else
		# Need first install from store to get all necessary assertions into
		# place. Second local install will then bring in our locally built
		# snap.
		snap install $name
		snap install --dangerous $PROJECT_PATH/$name*_amd64.snap
	fi
}

wait_for_systemd_service() {
	while ! systemctl status $1 ; do
		sleep 1
	done
	sleep 1
}

wait_for_udisksd() {
	wait_for_systemd_service snap.udisks2.udisksd
}

stop_after_first_reboot() {
	if [ $SPREAD_REBOOT -eq 1 ] ; then
		exit 0
	fi
}
