package go_mhda

import (
	"errors"
	"testing"
)

var (
	nssChainKey = []string{
		`nt:bitcoin:ci:bitcoin`, // Bitcoin
		`nt:tron:ci:mainnet`,    // Tron
		`nt:evm:ci:0x1`,         // Ethereum
		`nt:evm:ci:0xa86a`,      // Avalanche
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

// TestChainFromKey verifies the strict chain-key parser: a chain key is the
// bare identity "nt:<network>:ci:<chain_id>" and round-trips through Key().
func TestChainFromKey(t *testing.T) {
	for i := range nssChainKey {
		chain, err := ChainFromKey(ChainKey(nssChainKey[i]))
		if err != nil {
			t.Fatalf("ChainFromKey(%q): %v", nssChainKey[i], err)
		}
		if string(chain.Key()) != nssChainKey[i] {
			t.Fatalf("Key() = %q, want %q", chain.Key(), nssChainKey[i])
		}
		if chain.HasCoinType() {
			t.Fatalf("chain key %q must not carry coin-type metadata", nssChainKey[i])
		}
	}
}

// TestChainFromKeyRejectsCoinType ensures the pre-1.1 key format (with an
// embedded "ct" component) fails loudly with the dedicated sentinel instead
// of being silently reinterpreted.
func TestChainFromKeyRejectsCoinType(t *testing.T) {
	for _, key := range []string{
		`nt:evm:ct:60:ci:1`, // pre-1.1 canonical order
		`nt:evm:ci:1:ct:60`, // ct trailing
		`nt:bitcoin:ct:0:ci:bitcoin`,
	} {
		_, err := ChainFromKey(ChainKey(key))
		if !errors.Is(err, ErrCoinTypeInChainKey) {
			t.Errorf("ChainFromKey(%q): got %v, want ErrCoinTypeInChainKey", key, err)
		}
	}
}

// TestChainFromKeyRejectsExtraComponents ensures a chain key may not carry
// derivation, address-format or wallet components.
func TestChainFromKeyRejectsExtraComponents(t *testing.T) {
	for _, key := range []string{
		`nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`,
		`nt:evm:ci:1:aa:secp256k1`,
		`nt:evm:ci:1:wt:web3`,
		`nt:evm:ci:1:wi:5f2a8c31`,
	} {
		_, err := ChainFromKey(ChainKey(key))
		if !errors.Is(err, ErrInvalidChainKey) {
			t.Errorf("ChainFromKey(%q): got %v, want ErrInvalidChainKey", key, err)
		}
	}
}

// TestChainFromNSSAcceptsOptionalCoinType: the lenient NSS extractor accepts
// the optional ct metadata (in any position) and keeps it off the key.
func TestChainFromNSSAcceptsOptionalCoinType(t *testing.T) {
	for _, nss := range []string{
		`nt:evm:ci:1:ct:60`,
		`nt:evm:ct:60:ci:1`, // pre-1.1 component order still parses as NSS
		`nt:evm:ci:1:ct:60:dt:bip44:dp:m/44'/60'/0'/0/0`,
	} {
		chain, err := ChainFromNSS(nss)
		if err != nil {
			t.Fatalf("ChainFromNSS(%q): %v", nss, err)
		}
		if !chain.HasCoinType() || chain.CoinType() != 60 {
			t.Errorf("ChainFromNSS(%q): coin type not captured", nss)
		}
		if got := chain.String(); got != `nt:evm:ci:1` {
			t.Errorf("ChainFromNSS(%q): String() = %q, want %q", nss, got, `nt:evm:ci:1`)
		}
	}
}
