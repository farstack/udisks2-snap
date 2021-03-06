---
title: Automount
table_of_contents: True
---

# Automount

Automount is a feature which is useful when every removable storage device that
is plugged into a device, on which the udisks2 snap is running, should be
automatically mounted. This feature is known from traditional desktop operating
systems where a USB storage device is plugged in and a file explorer directly
opened to allow the user to explore the content of the device.

On devices running Ubuntu Core the udisks2 snap offers such a feature named
"automount". It watches for new USB storage devices being plugged into the
device and will directly mount them below the */media* directory.

When storage devices are automatically mounted below */media* every snap which
has a plug with the *removable-media* interface connected has access to the
content of the storage device.

The feature is turned off by default and will only respect devices which fulfill
a set of requirements. These are:

 * **Storage device is removable.** This can be a USB storage device or a CD/DVD put
   into a CD drive.
 * **Storage device is not a system device.** If the system is running from a
   removable device all partitions on that device are ignored as they contain
   sensitive data. If a snap would get access, for example, to the EFI boot
   partition, it could modify boot-relevant things and hijack the whole system.
 * **Storage device is not already mounted.** If the storage device is already
   mounted in */media* it will not be mounted again.

If all requirements are fulfilled by a storage device or some of its partitions,
they are mounted in */media* in a directory specific for the *root* user. The path
generally follows this schema: */media/<user>/<storage device id/name>*

When enabled this feature will also mount any device plugged in before the boot that
fulfill the above rules.

## Enable Automount

The udisks2 snap provides a single configuration option which can be used to turn
the automount feature on or off:

 * **automount.enable**

The option takes a boolean value. The meaning of the possible values are:

 * **true:** Enable automount feature.
 * **false (default):** Disable automount feature.

Changing the **automount.enable** configuration option takes immediate effect
and does not require a restart of the udisks2 service.

**Example:** Enable automount feature.

```
$ snap set udisks2 automount.enable=true
```

<br/>
**Example:** Disable automount feature.

```
$ snap set udisks2 automount.enable=false
```

## Enable Automount By Default From a Gadget snap

[Gadget snaps](https://docs.ubuntu.com/core/en/reference/gadget) can provide
default configuration for snaps. This allows a device to ship with automount
enabled by default. The necessary part which needs to be added to the
*gadget.yaml* file for this is

```
defaults:
  KTyDAsgNZg4u9UBJ63MHn6q12fmUjOfn:
      automount.enable: true
```

The cryptic key in the part above is the alpha-numeric id of the udisks2 snap
which is assigned from the Ubuntu Store. You can copy the full snippet into
your *gadget.yaml* as is.
