#!/bin/bash

# We don't have to build a snap when we should use one from a
# channel
if [ -n "$SNAP_CHANNEL" ] ; then
	exit 0
fi

# If there is a udisks2 snap prebuilt for us, lets take
# that one to speed things up.
if [ -e /home/udisks2/udisks2_*_amd64.snap ] ; then
	exit 0
fi

# Setup classic snap and build the udisks2 snap in there
snap install --devmode --beta classic
cat <<-EOF > /home/test/build-snap.sh
#!/bin/sh
set -ex

export DEBIAN_FRONTEND=noninteractive

# FIXME: Enable propose for now until problems with conflicting systemd
# packages between the Ubuntu Core image ppa and the archive are fixed.
echo "deb http://archive.ubuntu.com/ubuntu/ xenial-proposed restricted main universe" > /etc/apt/sources.list.d/ubuntu-proposed.list

# Ensure we have the latest updates installed as the core snap
# may be a bit out of date.
apt update
apt full-upgrade -y --force-yes -o Dpkg::Options::="--force-confold"

apt install -y --force-yes snapcraft
cd /home/udisks2
snapcraft clean
snapcraft
EOF
chmod +x /home/test/build-snap.sh
sudo classic /home/test/build-snap.sh
snap remove classic

# Make sure we have a snap build
test -e /home/udisks2/udisks2_*_amd64.snap
