## Vulnerability report on IOTA and code to create collisions

Read the original report [here](vuln-iota.md).

See `examples` for the original colliding bundles we released in 2017.

See `valueattack`, `collide`, and `template` for the code to create colliding bundles.

Make sure to set your `GOPATH` and check out this repo to `$GOPATH/src/github.com/mit-dci/tangled-curl`

```
`cd $GOPATH/src/github.com/mit-dci/tangled-curl/valueattack`
go get -u github.com/getlantern/deepcopy
go get -u github.com/iotaledger/giota
```

Make sure iotaledger is at the right commit:

```
pushd $GOPATH/src/github.com/iotaledger/giota/
git checkout 7e48a1c9b9e904f07e1fc82815e5b302873a6dec
popd
go build
```


