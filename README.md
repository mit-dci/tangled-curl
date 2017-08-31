## Colliding IOTA Bundles

Example of valid IOTA bundles which collide on the 72nd trit of the Address field
of the last transaction in each bundle.  The bundles have the same hash, and
thus the same signature.

```
$ go build
$ ./tangled-curl
Collision! Can burn funds
Collision! Can steal funds
$
```