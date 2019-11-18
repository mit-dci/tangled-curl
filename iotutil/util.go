package iotutil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/iotaledger/giota"
	"github.com/mit-dci/tangled-curl/collide"
)

// ReadBundle reads the given file from the filesystem and unmarshals
// it into a Go type. Expects a JSON array of trytes in string format.
func ReadBundle(filename string) ([]string, error) {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Error reading file %v\n", filename)
		return nil, err
	}

	var bundle []string

	err = json.Unmarshal(dat, &bundle)
	if err != nil {
		log.Printf("Error unmarshaling %v\n", filename)
		return nil, err
	}

	return bundle, nil
}

// Creates a bundle from a file of tryte strings.
func BundleFromTrytesFile(filename string) (giota.Bundle, error) {
	bundle, err := ReadBundle(filename)
	if err != nil {
		return nil, err
	}

	var bundleTrytes []giota.Trytes

	for _, tx := range bundle {
		bundleTrytes = append(bundleTrytes, giota.Trytes(tx))
	}

	return bundleFromTrytes(bundleTrytes), nil
}

// Writes a bundle to a file as a JSON array of strings.
func BundleToTrytesFile(b giota.Bundle, filename string) error {
	var trytes []string
	for i := 0; i < len(b); i++ {
		trytes = append(trytes, string(b[i].Trytes()))
	}
	dat, err := json.Marshal(trytes)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, dat, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Converts a bundle to trits.
func BundleToTrits(bundle giota.Bundle) []int8 {
	ret_trits := make([]int8, one_sz*len(bundle))
	for i := 0; i < len(bundle); i++ {
		trits := txnToTrits(bundle[i])
		copy(ret_trits[one_sz*i:one_sz*(i+1)], trits)
	}
	return ret_trits
}

func TxnFromMB(mb int) int {
	return mb / 2
}

func CmpSlice(a, b []int8, stop bool) bool {
	if len(a) != len(b) {
		fmt.Println("Unequal sizes")
		return false
	}
	diff := false
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			fmt.Printf("diff: %v, a[%v]=%v, b[%v]=%v\n", i, i, a[i], i, b[i])
			diff = true
			if stop {
				return false
			}
		}
	}
	if diff {
		return false
	} else {
		return true
	}
}

func PrintBundle(bundle giota.Bundle) {
	for i := 0; i < len(bundle); i++ {
		b := bundle[i]
		fmt.Printf("  ")
		printtxn(b)
	}
}

// Do proof-of-work so the transaction is accepted into the main
// tangle.  This is not a fee, despite the fact that it takes multiple
// minutes on my computer. Sure.
//
// Set up the branch and trunk transactions as well.
func DoPow(bundle giota.Bundle) {
	var prev giota.Trytes
	var err error
	// Point to any old index 0 transaction
	var tra giota.Trytes = giota.Trytes(strings.Repeat("9", 81))

	for i := len(bundle) - 1; i >= 0; i-- {
		if i == len(bundle)-1 {
			bundle[i].TrunkTransaction = tra
			bundle[i].BranchTransaction = tra
		} else {
			bundle[i].TrunkTransaction = prev
			bundle[i].BranchTransaction = tra
		}
		bundle[i].Nonce, err = giota.PowGo(bundle[i].Trytes(), 15) // Sergey says 15 for mainnet
		if err != nil {
			panic(err)
		}
		prev = bundle[i].Hash()
	}
}

func Sign(seed giota.Trytes, bundle giota.Bundle) {
	ais := make([]giota.AddressInfo, 1)
	ais[0] = giota.AddressInfo{seed, 1, 2}

	bundle.Finalize(nil)
	signInputs(ais, bundle)
}

// The Go code doesn't check as much as the Java code.  Reproduce
// some checks and call the Go code.
//
// Does not handle multisig (TODO: but it should!)
func Validate(txns giota.Bundle) bool {
	LightValidate(txns)
	if err := txns.IsValid(); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

// The Go code doesn't check as much as the Java code.  Reproduce
// some checks here.
//
// Does not check signatures or PoW
func LightValidate(txns giota.Bundle) bool {
	var total int64
	for i, b := range txns {
		total += b.Value
		if b.CurrentIndex != int64(i) {
			fmt.Printf("CurrentIndex of index %d is not correct", b.CurrentIndex)
			return false
		}
		if b.LastIndex != int64(len(txns)-1) {
			fmt.Printf("LastIndex of index %d is not correct", b.CurrentIndex)
			return false
		}
		// check timestamp not too early. Note IOTA uses 1502226000
		// after the upgrade; this bundle is pre-upgrade.
		if txns[i].Timestamp.Unix() < 1497031200 {
			fmt.Printf("Invalid transaction %v timestamp %v\n", i, txns[i].Timestamp)
			return false
		}
		// check value only in 33 trits:
		// https://github.com/iotaledger/iri/blob/dev/src/main/java/com/iota/iri/TransactionValidator.java#L77
		vtrits := giota.Int2Trits(txns[i].Value, 81)
		for t := 34; t < 81; t++ {
			if vtrits[t] != 0 {
				fmt.Printf("Invalid transaction %v value %v\n", i, txns[i].Value)
				return false
			}
		}
	}
	if total != 0 {
		fmt.Printf("total balance of Bundle is not 0")
		return false
	}
	return true
}

func CmpBundle(b0 giota.Bundle, b1 giota.Bundle) (map[int]bool, bool) {
	if len(b0) != len(b1) {
		fmt.Printf("Unequal # txns %v %v\n", len(b0), len(b1))
		return nil, false
	}
	m := make(map[int]bool)
	diff := false
	for i := 0; i < len(b0); i++ {
		if b0[i].CurrentIndex != b1[i].CurrentIndex {
			fmt.Printf("Bundles differ at Current index %v\n", i)
			diff = true
			m[i] = true
		}
		if b0[i].LastIndex != b1[i].LastIndex {
			fmt.Printf("Bundles differ at Last index %v\n", i)
			diff = true
			m[i] = true
		}
		if b0[i].Address != b1[i].Address {
			fmt.Printf("Bundles differ at Address %v\n", i)
			diff = true
			m[i] = true
		}
		if b0[i].Tag != b1[i].Tag {
			fmt.Printf("Bundles differ at Tag %v\n", i)
			diff = true
			m[i] = true
		}
		if b0[i].Timestamp != b1[i].Timestamp {
			fmt.Printf("Bundles differ at Timestamp index %v b0.Timestamp=%v b1.Timestamp=%v\n", i, b0[i].Timestamp, b1[i].Timestamp)
			diff = true
			m[i] = true
		}
		if b0[i].Value != b1[i].Value {
			fmt.Printf("Bundles have different values, index %v.\n\tb0.Value=%v\n\tb1.Value=%v\n", i, b0[i].Value, b1[i].Value)
			diff = true
			m[i] = true
		}
	}
	if diff {
		CmpSlice(BundleToTrits(b0), BundleToTrits(b1), false)
	}
	return m, !diff
}

// Debug helper function when finding collisions
func DebugBundles(b0, b1 giota.Bundle, msg0, msg1, state3 []int8) {
	fmt.Printf("Comparing bundles: ")
	fmt.Printf("Value, txn 1. b0.Value=%v b1.Value=%v\n", b0[1].Value, b1[1].Value)
	CmpBundle(b0, b1)
	bt0 := BundleToTrits(b0)
	bt1 := BundleToTrits(b1)
	if !CmpSlice(bt0[3*243:4*243], msg0, false) {
		fmt.Printf("mb3 is different than msg0 returned\n")
	}
	if !CmpSlice(bt1[3*243:4*243], msg1, false) {
		fmt.Printf("mb3 is different than msg1 returned\n")
	}
	for i := 0; i < 18; i++ {
		fmt.Printf("Absorbing and checking mb %v...\n", i)
		s0 := collide.ProduceState(bt0, i)
		s1 := collide.ProduceState(bt1, i)
		if i == 2 {
			if !CmpSlice(s0, state3, true) {
				fmt.Printf("State after absorbing mb%v is different than previous state3 in b0\n", i)
			}
			if !CmpSlice(s1, state3, true) {
				fmt.Printf("State after absorbing mb%v is different than previous state3 in b1\n", i)
			}
		}
	}
	fmt.Println("Bundle0:")
	PrintBundle(b0)
	fmt.Println("Bundle1:")
	PrintBundle(b1)
}

func bundleFromTrytes(trytes []giota.Trytes) giota.Bundle {
	txns := make([]giota.Transaction, len(trytes))

	for i := 0; i < len(trytes); i++ {
		tx, err := giota.NewTransaction(trytes[i])
		if err != nil {
			fmt.Printf("Panic on new txn\n")
			panic(err)
		}
		txns[tx.CurrentIndex] = *tx
	}
	return giota.Bundle(txns)
}

var one_sz int = 243 + 81*2 + 27*3

func txnToTrits(tx giota.Transaction) []int8 {
	trits := make([]int8, 243+81+81+27+27+27)
	copy(trits[0:243], giota.Trytes(tx.Address).Trits())
	copy(trits[243:243+81], giota.Int2Trits(tx.Value, 81))
	copy(trits[243+81:243+81+81], tx.Tag.Trits())
	copy(trits[243+81+81:243+81+81+27], giota.Int2Trits(tx.Timestamp.Unix(), 27))
	copy(trits[243+81+81+27:243+81+81+27+27], giota.Int2Trits(tx.CurrentIndex, 27))
	copy(trits[243+81*2+27*2:243+81*2+27*3], giota.Int2Trits(tx.LastIndex, 27))
	return trits
}

func bundleHashDataFromTrits(trits []int8) giota.Bundle {
	if len(trits)%486 != 0 {
		panic("Bad length")
	}
	num := len(trits) / 486
	b := make([]giota.Transaction, num)
	for i := 0; i < num; i++ {
		var err error
		t := trits[i*486 : (i+1)*486]
		b[i].Address, err = giota.Trits(t[0:243]).Trytes().ToAddress()
		if err != nil {
			panic(err)
		}
		b[i].Value = giota.Trits(t[243 : 243+81]).Int()
		b[i].Tag = giota.Trits(t[243+81 : 243+81+81]).Trytes()
		ts := giota.Trits(t[243+81+81 : 243+81+81+27]).Int()
		b[i].Timestamp = time.Unix(ts, 0)
		b[i].CurrentIndex = giota.Trits(t[243+81+81+27 : 243+81+81+27+27]).Int()
		b[i].LastIndex = giota.Trits(t[243+81+81+27+27 : 243+81+81+27+27+27]).Int()
	}
	return giota.Bundle(b)
}

func printtxn(b giota.Transaction) {
	fmt.Printf("%v %v\t %v\t %v\t %v\t (%v)\n", b.CurrentIndex, b.LastIndex, b.Address, b.Tag, b.Timestamp, b.Value)
}

//
// IOTA code
// This is a copy of the IOTA go code made available under the MIT License @ https://github.com/iotaledger/giota
//

/*
MIT License
Copyright (c) 2016 Sascha Hanse
Copyright (c) 2017 Shinya Yagyu
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

func signInputs(inputs []giota.AddressInfo, bundle giota.Bundle) error {
	//  Get the normalized bundle hash
	nHash := bundle.Hash().Normalize()

	//  SIGNING OF INPUTS
	//
	//  Here we do the actual signing of the inputs
	//  Iterate over all bundle transactions, find the inputs
	//  Get the corresponding private key and calculate the signatureFragment
	for i, bd := range bundle {
		if bd.Value >= 0 {
			continue
		}
		// Get the corresponding keyIndex and security of the address
		var ai giota.AddressInfo
		for _, in := range inputs {
			adr, err := in.Address()
			if err != nil {
				return err
			}
			if adr == bd.Address {
				ai = in
				break
			}
		}
		// Get corresponding private key of address
		key := ai.Key()
		//  Calculate the new signatureFragment with the first bundle fragment
		bundle[i].SignatureMessageFragment = giota.Sign(nHash[:27], key[:6561/3])

		// if user chooses higher than 27-tryte security
		// for each security level, add an additional signature
		for j := 1; j < ai.Security; j++ {
			//  Because the signature is > 2187 trytes, we need to
			//  find the subsequent transaction to add the remainder of the signature
			//  Same address as well as value = 0 (as we already spent the input)
			if bundle[i+j].Address == bd.Address && bundle[i+j].Value == 0 {
				//  Calculate the new signature
				nfrag := giota.Sign(nHash[(j%3)*27:(j%3)*27+27], key[6561*j/3:(j+1)*6561/3])
				//  Convert signature to trytes and assign it again to this bundle entry
				bundle[i+j].SignatureMessageFragment = nfrag
			}
		}
	}
	return nil
}
