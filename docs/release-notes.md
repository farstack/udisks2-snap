---
title: "Release Notes"
table_of_contents: False
---

# Release Notes

The version numbers mentioned on this page correspond to those released in the
Ubuntu snap store.

You can check with the following command which version you have currently
installed:

```
$ snap info udisks2
name:      udisks2
summary:   "D-Bus service to access and manipulate storage devices"
publisher:
description: |
  The udisks project provides a daemon, tools and libraries to access
  and manipulate disks and storage devices.

  Please find the source code for this snap at:
  https://code.launchpad.net/~snappy-hwe-team/snappy-hwe-snaps/+git/udisks

commands:
  - udisks2.udisksctl (udisksctl)
tracking:    stable
installed:   2.1.7-2 (9)  3MB -
refreshed:   2017-03-07 11:21:42 +0000 UTC
channels:
  stable:    2.1.7-2 (9)  2MB -
[...]
```
</br>

## 2.1.7-6

 * Bug fix release to prevent udisks2 from automatically mounting system critical
   boot partitions into /media.

## 2.1.7-5

 * Added regression test case to verify that a huge number of mount entries
   doesn't cause ciborium to fail reading dbus messages

## 2.1.7-4

 * Initial automount support which can be enabled via a snap configuration option

## 2.1.7-2

 * Initial release of the udisks2 snap
