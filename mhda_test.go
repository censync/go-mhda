package go_mhda

import "testing"

var (
	uriMHDA = []string{
		`urn:mhda:nt:evm:dt:bip44:dp:m/44'/60'/1'/0/1:ct:60:ci:1:aa:secp256k1:af:hex:ap:0x`,
		`urn:mhda:nt:evm:dt:bip44:dp:m/44'/60'/2'/0/2':ct:60:ci:1`,
		`urn:mhda:nt:evm:ct:60:ci:1`,
		`urn:mhda:nt:btc:dt:bip44:dp:m/44'/0'/0'/0/0:ct:0:ci:bitcoin_testnet`,
		`urn:mhda:nt:btc:dt:bip44:dp:m/44'/0'/1'/0/1:ct:0:ci:bitcoin:aa:secp256k1:af:p2pkh:ap:1`,
		//`urn:mhda:nt:btc:dt:bip84:dp:m/84'/0'/2'/0/2:ct:0:ci:bitcoin:aa:secp256k1:af:p2pkh:ap:bc1q`,
	}
)

func TestParse(t *testing.T) {
	for i := 0; i < len(uriMHDA); i++ {
		addr, err := ParseURN(uriMHDA[i])
		t.Log(addr.String())
		//t.Log(addr.Hash())
		//t.Log(addr.NSSHash())
		if err != nil {
			t.Fatal(err)
		}

		/*if addr.String() != uriMHDA[i] {
			t.Fatal("mismatch result", addr.String(), uriMHDA[i])
		}*/
	}
}

func TestParseNSS(t *testing.T) {
	for i := 0; i < len(uriMHDA); i++ {
		t.Log(uriMHDA[i][prefixOffset:])
		addr, err := ParseURN(uriMHDA[i])
		if err != nil {
			t.Fatal(err)
		}
		t.Log(err, addr.NSS())
	}
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseURN(uriMHDA[0])
	}
}

func BenchmarkParseRx(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseURNRx(uriMHDA[0])
	}
}
