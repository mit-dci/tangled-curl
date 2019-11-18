## Vulnerability report on IOTA and code to create collisions

Read our full paper [Cryptanalysis of Curl-P and Other Attacks on the IOTA Cryptocurrency](http://i.blackhat.com/us-18/Wed-August-8/us-18-Narula-Heilman-Cryptanalysis-of-Curl-P-wp.pdf).

Read the original report [here](vuln-iota.md).

See `examples` for the original colliding bundles we released in 2017.

See `valueattack`, `collide`, and `template` for the code to create colliding bundles.

Make sure to set your `GOPATH` and check out this repo to `$GOPATH/src/github.com/mit-dci/tangled-curl`. For example, the following sets `GOPATH` to a directory named `go` inside your home directory and clones the repo there:

```
export GOPATH=$HOME/go
mkdir -p $GOPATH/src/github.com/mit-dci
cd $GOPATH/src/github.com/mit-dci
git clone https://github.com/mit-dci/tangled-curl
```

Afterwards, clone the IOTA libraries:

```
go get -u github.com/getlantern/deepcopy
go get -u github.com/iotaledger/giota
```

The latter line will emit a harmless warning (`package github.com/iotaledger/giota: no Go files in ...`). As `iotaledger` changed the implementation since we wrote our cryptanalysis code, make sure that `iotaledger` is at the right commit:

```
pushd $GOPATH/src/github.com/iotaledger/giota/
git checkout 7e48a1c9b9e904f07e1fc82815e5b302873a6dec
popd
```

Install pypy (our code hardcodes `pypy` executable name but it is likely that `pypy3` would work with small changes).

Finally, try out our attack:

```
cd $GOPATH/src/github.com/mit-dci/tangled-curl/valueattack
CGO_LDFLAGS_ALLOW='-msse2' go build
./valueattack
```

(The `CGO_LDFLAGS_ALLOW` environment variable enables [cgo flag whitelisting](https://github.com/golang/go/wiki/InvalidFlag) required by iotaledger at the commit we use.)
