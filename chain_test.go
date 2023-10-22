package go_mhda

import "testing"

var (
	nssChainKey = []string{
		`nt:btc:ct:0:ci:bitcoin`,   // Bitcoin
		`nt:tvm:ct:195:ci:mainnet`, // Tron
		`nt:evm:ct:60:ci:0x1`,      // Ethereum
		`nt:evm:ct:60:ci:0xa86a`,   // Avalanche
	}
)

func TestChainFromNSS(t *testing.T) {
	for i := range nssChainKey {
		chain, err := ChainFromNSS(nssChainKey[i])
		if err != nil {
			t.Fatalf("Cannot parse %s", nssChainKey[i])
		}

		if chain.String() != nssChainKey[i] {
			t.Fatalf(
				"Unmatched parsed chain key \"%s\" vs \"%s\"",
				chain.String(),
				nssChainKey[i],
			)
		}
	}
}
