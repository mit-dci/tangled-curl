package main

import (
	"fmt"

	"github.com/iotaledger/giota"
)

func bundleFromTrytes(trytes []string) giota.Bundle {
	txns := make([]giota.Transaction, len(trytes))
	for i := 0; i < len(trytes); i++ {
		tx, err := giota.NewTransaction(giota.Trytes(trytes[i]))
		if err != nil {
			panic(err)
		}
		txns[i] = *tx
	}
	return giota.Bundle(txns)
}

func validate(txns giota.Bundle) bool {
	h := txns.Hash()
	for i := 0; i < len(txns); i++ {
		// The golang code doesn't check as much as the java code.
		// Reproduce some checks here.

		// Check to make sure bundle hashes are valid
		if txns[i].Bundle != h {
			fmt.Printf("Invalid transaction %v bundle hash %v\n", i, txns[i].Bundle)
			return false
		}

		// check timestamp not too early. Note IOTA uses 1502226000
		// after the upgrade; this bundle is pre-upgrade.
		if txns[i].Timestamp.Unix() < 1497031200 {
			fmt.Printf("Invalid transaction %v timestamp %v\n", i, txns[i].Timestamp)
			return false
		}
		// check nonce (proof of work; main tangle requires >= 15)
		if !txns[i].HasValidNonce(15) {
			fmt.Printf("Invalid transaction %v nonce for weight 15 %v\n", i, txns[i].Nonce)
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
	if err := txns.IsValid(); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func main() {
	bundle0 := bundleFromTrytes(BURN_BUNDLE0)
	bundle1 := bundleFromTrytes(BURN_BUNDLE1)
	if !validate(bundle0) || !validate(bundle1) {
		return
	}

	if bundle0.Hash() == bundle1.Hash() && bundle0[3].Address != bundle1[3].Address {
		fmt.Println("Collision! Can burn funds")
	}

	bundle0 = bundleFromTrytes(STEAL_BUNDLE0)
	bundle1 = bundleFromTrytes(STEAL_BUNDLE1)
	if !validate(bundle0) || !validate(bundle1) {
		return
	}

	if bundle0.Hash() == bundle1.Hash() && bundle0[5].Value != bundle1[5].Value {
		fmt.Println("Collision! Can steal funds")
	}

}
