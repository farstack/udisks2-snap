---
title: Debug
table_of_contents: True
---

# Debug

Debug is a feature that controls the amount of logs produced by the udisks2
snap. It is useful for collecting information required to either report a bug or
investigate a udisks2 failure (if happens).

The default is disabled by default and has to be explicitely turned on for
usage.

## Enable Debug

The udisks2 snap provides a single configuration option which can be used to
turn the debug feature either on or off:

 * **debug.enable**

The option takes a boolean value. The meaning of the possible values are:

 * **true:** Enable logging debug information
 * **false (default):** Disable logging debug information

Chnaging the **debug.enable** configuration option does not have immediate
effect and require a restart of the ciborium service.

**Example:** Enable debug feature

```
$ snap set udisks2 debug.enable=true
```

<br/>
**Example:** Disable debug feature.

```
$ snap set udisks2 debug.enable=false
```

