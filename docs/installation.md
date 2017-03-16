---
title: "Install udisks2"
table_of_contents: True
---

# Install udisks2

The udisks2 snap is currently available from the Ubuntu Store. It can
be installed on any system that supports snaps but is only recommended on
[Ubuntu Core](https://www.ubuntu.com/core) at the moment.

You can install the snap with the following command:

```
 $ snap install udisks2
 udisks2 2.1.7-6 from 'canonical' installed
```

Although the udisks2 snap is available from other channels (candidate, beta, edge),
only the stable version should be used for production devices. Their meaning is internal
to the development team of the udisks2 snap.

All necessary plugs and slots will be automatically connected within the
installation process. You can verify this with:

```
$ snap interfaces udisks2
Slot             Plug
:mount-observe   udisks2
:network-bind    udisks2
udisks2:service  udisks2:client
-                udisks2:hardware-observe
```

**NOTE:** The _udisks2:hardware-observe_ plug currently isn't automatically
connected and will be either with a future version or dropped from the udisks2
snap. Don't worry about it.

Once the installation has successfully finished the udisks2 service should be
running in the background. You can check its current status with

```
 $ sudo systemctl status snap.udisks2.udisksd
 ● snap.udisks2.udisksd.service - Service for snap application udisks2.udisksd
    Loaded: loaded (/etc/systemd/system/snap.udisks2.udisksd.service; enabled; vendor preset: enabled)
    Active: active (running) since Wed 2017-03-08 08:10:33 UTC; 4h 26min ago
  Main PID: 1619 (udisksd)
    CGroup: /system.slice/snap.udisks2.udisksd.service
            └─1619 /snap/udisks2/x2/libexec/udisks2/udisksd
```

Now you have udisks2 successfully installed.

## Next Steps

 * [Enable Automount Support](reference/snap-configuration/automount.md)
