# Udisks

This is the snap package for the Udisks storage management service.

## Automount

This snap is capable of automounting removable drives, but this capability is turned off by default. If you wish to enable it, run

    $ snap set udisks2 automount=on

To disable it run

    $ snap set udisks2 automount=off
