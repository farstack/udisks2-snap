# Udisks

This is the snap package for the Udisks storage management service.

## Automount

This snap is capable of automounting removable drives, but this capability is turned off by default. If you wish to enable it, run

    $ snap set udisks2 automount.enable=true

To disable it run

    $ snap set udisks2 automount.enable=false
