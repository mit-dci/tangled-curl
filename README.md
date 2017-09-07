## Vulnerability report on IOTA and colliding bundles

Read the report [here](vuln-iota.md).

Examples of valid IOTA bundles which collide.

BURN_BUNDLEs collide on the 72nd trit of the Address field of the last
transaction in each bundle.

STEAL_BUNDLEs collide on the 17th trit of the Value fields in the 4th
and 6th transaction in each bundle.

The bundles in each pair have the same hash, and thus the same
signature.

```
$ go build
$ ./tangled-curl
Collision! Can burn funds
Collision! Can steal funds
$
```