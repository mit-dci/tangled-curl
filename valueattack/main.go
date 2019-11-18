package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/getlantern/deepcopy"
	"github.com/iotaledger/giota"
	"github.com/mit-dci/tangled-curl/collide"
	"github.com/mit-dci/tangled-curl/iotutil"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var bundlefn = flag.String("fn", "inputBundleTrytes.txt", "Bundle filename")
var trit = flag.Int("trit", 26, "Trit to collide")
var ncpu = flag.Int("ncpu", 4, "Number of goroutines")
var col1 = flag.Int("col1", 3, "Message block index of first collision")
var col2 = flag.Int("col2", 7, "Message block index of second collision")
var nitr = flag.Int("nitr", 1, "Number of iterations")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	min := time.Hour
	max := time.Nanosecond
	start := time.Now()
	cmd := "pypy ../template/trit17.py"

	for nt := 0; nt < *nitr; nt++ {

		b0, err := iotutil.BundleFromTrytesFile(*bundlefn)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		if *col1 < 3 || *col2 > 2*len(b0)-1 {
			log.Fatal("Collisions cannot be in first two or last blocks")
		}
		if *col1%2 == 0 || *col2%2 == 0 {
			log.Fatal("We assume collisons in the Value field; choose an odd message block")
		}

		if *trit != 17 && *trit != 26 {
			log.Fatal("For now we only handle collisions on trits 17 or 26")
		}
		if *trit == 26 {
			cmd = "pypy ../template/trit26.py"
		}

		trits := iotutil.BundleToTrits(b0)

		start_internal := time.Now()
		// tx0 value and tag
		mb1 := trits[(*col1-2)*243 : (*col1-1)*243]
		// tx1 addr
		mb2 := trits[(*col1-1)*243 : (*col1)*243]
		// tx1 value and tag
		mb3 := trits[(*col1)*243 : (*col1+1)*243]

		// Transform addr of tx0
		state := collide.ProduceState(trits, *col1-3)

		// Give this state to python
		tag_trits, template := collide.RunPython(state, mb1, mb2, mb3, cmd)

		// Set up the first tag, look for collisions
		tag := giota.Trits(tag_trits[81 : 81+81]).Trytes()
		b0[iotutil.TxnFromMB(*col1-2)].Tag = tag

		trits = iotutil.BundleToTrits(b0)

		state = collide.ProduceState(trits, *col1-1)

		fmt.Println("Generating collision...")
		msg0, msg1, _ := collide.Collide(state, template, *ncpu, 1, 6)

		// Found first collision
		b0[iotutil.TxnFromMB(*col1)].Tag = giota.Trits(msg0[81 : 81+81]).Trytes()

		// Now we need two bundles.
		var b1 giota.Bundle
		if err := deepcopy.Copy(&b1, &b0); err != nil {
			fmt.Printf("Panic on deepcopy\n")
			panic(err)
		}

		// Set the value in the colliding bundle to have a 1 in the 17th trit
		b0[iotutil.TxnFromMB(*col1)].Value = giota.Trits(msg0[0:81]).Int()
		b1[iotutil.TxnFromMB(*col1)].Value = giota.Trits(msg1[0:81]).Int()

		if b0.Hash() != b1.Hash() {
			fmt.Printf("wrong hashes on 1st collision:\n  b0: %v\n  b1: %v\n", b0.Hash(), b1.Hash())
			return
		}
		fmt.Println("First collision worked!")

		// Run again for the second collision
		trits = iotutil.BundleToTrits(b0)

		// tx7 value and tag
		mb1 = trits[(*col2-2)*243 : (*col2-1)*243]
		// tx8 addr
		mb2 = trits[(*col2-1)*243 : (*col2)*243]
		// tx8 value and tag
		mb3 = trits[(*col2)*243 : (*col2+1)*243]

		state = collide.ProduceState(trits, (*col2 - 3))

		// Give this state to python
		tag_trits, template = collide.RunPython(state, mb1, mb2, mb3, cmd)

		// Set up the next tag, look for collisions
		tag = giota.Trits(tag_trits[81 : 81+81]).Trytes()
		b0[iotutil.TxnFromMB(*col2-2)].Tag = tag
		b1[iotutil.TxnFromMB(*col2-2)].Tag = tag

		trits = iotutil.BundleToTrits(b0)
		state = collide.ProduceState(trits, *col2-1)

		fmt.Println("Generating second collision...")
		msg0, msg1, _ = collide.Collide(state, template, *ncpu, 1, 6)

		// Found second collision
		b0[iotutil.TxnFromMB(*col2)].Tag = giota.Trits(msg0[81 : 81+81]).Trytes()
		b1[iotutil.TxnFromMB(*col2)].Tag = giota.Trits(msg1[81 : 81+81]).Trytes()

		// Set the value in the colliding bundle to have a 1 in the 17th or 26th trit
		b0[iotutil.TxnFromMB(*col2)].Value = giota.Trits(msg1[0:81]).Int()
		b1[iotutil.TxnFromMB(*col2)].Value = giota.Trits(msg0[0:81]).Int()

		if b0.Hash() != b1.Hash() {
			fmt.Printf("wrong hashes on 2nd collision:\n  b0: %v\n  b1: %v\n", b0.Hash(), b1.Hash())
			return
		}
		fmt.Println("Second collision!")

		if b0.Hash() != b1.Hash() {
			log.Fatalf("wrong hashes")
		}
		fmt.Printf("Bundles have the same hash!\n")

		if !iotutil.LightValidate(b0) || !iotutil.LightValidate(b1) {
			log.Fatalf("invalid")
		}

		x := time.Since(start_internal)
		if x > max {
			max = x
		}
		if x < min {
			min = x
		}

		if nt == 0 {
			fmt.Printf("Bundle 0 tags and values: \n")
			for i := 0; i < len(b0); i++ {
				fmt.Printf("\t%v\t%v\n", b0[i].Tag, b0[i].Value)
			}
			fmt.Printf("Bundle 1 tags and values: \n")
			for i := 0; i < len(b1); i++ {
				fmt.Printf("\t%v\t%v\n", b1[i].Tag, b1[i].Value)
			}
			err := iotutil.BundleToTrytesFile(b0, "b0.json")
			if err != nil {
				log.Fatalf("b0 write to file: %v", err)
			}
			err = iotutil.BundleToTrytesFile(b1, "b1.json")
			if err != nil {
				log.Fatalf("b1 write to file: %v", err)
			}
		} else {
			log.Printf("itr: %v, %v average time\n", nt, time.Since(start)/time.Duration(nt))
		}
	}
	if *nitr > 1 {
		log.Printf("%v average time, min: %v, max: %v\n", time.Since(start)/time.Duration(*nitr), min, max)
	} else {
		log.Printf("%v time\n", time.Since(start)/time.Duration(*nitr))
	}
}
