---
title: "Report a Bug"
table_of_contents: False
---

# Report a Bug

Bugs can be reported [here](https://bugs.launchpad.net/snappy-hwe-snaps/+filebug).

When submitting a bug report, please first enable logging by setting the 
**debug.enabled** snao option to **true** and restarting the ciborium service.

```
$ sudo snap set udisks2 debug.enable=true
$ sudo systemctl restart snap.udisks2.ciborium.service
```

Now repeat the steps that lead to the failure and, please attach:

 * */var/log/syslog*

And the output of the following two commands:

```
$ sudo udisksctl dump
$ sudo udisksctl status
```

If you have problems with a particular storage device please also attach the
output of

```
$ sudo udevadm info /dev/<storage device><n>
```

For example, if your storage device becomes available as */dev/sdb* and has two
partitions */dev/sdb1* and */dev/sdb2* you need to add the output of the following
two commands

```
$ sudo udevadm info /dev/sdb
$ sudo udevadm info /dev/sdb1
$ sudo udevadm info /dev/sdb2
```
