# go-mhda

MultiChain Hierarchical Deterministic Address (MHDA) format - a URN-based
descriptor for blockchain HD addresses, with [RFC 8141](https://datatracker.ietf.org/doc/rfc8141/)
compatibility.

A single string captures everything needed to identify a derived address:
network, derivation scheme, path, signature curve, encoding format, any
prefix/suffix conventions, and an optional wallet context.

```
urn:mhda:nt:bitcoin:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m:ap:bc1p
```

Supported networks: Bitcoin, EVM, Avalanche, Tron, Cosmos, Solana, XRP Ledger,
Stellar, NEAR, Aptos, Sui, Cardano, Algorand, TON.

The chain identity is the `(nt, ci)` pair; the SLIP-44 coin type is an
optional `ct` metadata component (for HD addresses the coin already lives in
the derivation path). The optional wallet domain (`wt`/`wi`) binds an address
to a client type and a wallet instance:

```
urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0:wt:web3:wi:5f2a8c31
```

## Installation

```sh
go get github.com/censync/go-mhda
```

## Quick start

```go
import (
    "encoding/json"
    "errors"
    "fmt"

    mhda "github.com/censync/go-mhda"
)

func main() {
    // Lenient parsing: structural validation only.
    addr, err := mhda.ParseURN("urn:mhda:nt:evm:ci:1")
    if err != nil {
        panic(err)
    }
    fmt.Println(addr.Chain().NetworkType()) // evm
    fmt.Println(addr.Algorithm())           // secp256k1 (default for evm)
    fmt.Println(addr.Format())              // hex (default for evm)

    // Strict parsing: also checks the (network, algorithm, format,
    // derivation) combination is in the known-good compatibility matrix.
    if _, err := mhda.ParseURNStrict("urn:mhda:nt:evm:ci:1:aa:ed25519"); err != nil {
        if errors.Is(err, mhda.ErrIncompatible) {
            fmt.Println("evm + ed25519 rejected, as expected")
        }
    }

    // JSON / YAML / text codecs work out of the box.
    raw, _ := json.Marshal(addr)
    var back mhda.Address
    _ = json.Unmarshal(raw, &back)

    // Type-agnostic level-by-level path inspection.
    for i, lvl := range addr.DerivationPath().Levels() {
        fmt.Printf("level[%d] = %d (hardened=%v)\n", i, lvl.Index, lvl.IsHardened)
    }

    // Hashing for content-addressing or deduplication.
    fmt.Println(addr.Hash256())   // SHA-256, hex

    // The chain identity (nt, ci) is itself a parseable key. Keys never
    // carry the optional ct metadata; a pre-1.1 key with ct fails loudly
    // with ErrCoinTypeInChainKey.
    key := addr.Chain().Key() // "nt:evm:ci:1"
    chain, _ := mhda.ChainFromKey(key)
    _ = chain

    // Optional wallet context: client type + wallet instance id.
    wallet, _ := mhda.ParseURN("urn:mhda:nt:evm:ci:1:wt:web3:wi:5f2a8c31")
    fmt.Println(wallet.WalletType(), wallet.WalletId()) // web3 5f2a8c31
}
```

## Examples

| Network        | Example URN                                                                                          |
|----------------|------------------------------------------------------------------------------------------------------|
| Ethereum       | `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`                                                  |
| Bitcoin (BIP86)| `urn:mhda:nt:bitcoin:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m:ap:bc1p`                      |
| Solana         | `urn:mhda:nt:solana:ci:mainnet:dt:slip10:dp:m/44'/501'/0'/0'`                                        |
| Stellar        | `urn:mhda:nt:stellar:ci:mainnet:dt:slip10:dp:m/44'/148'/0'`                                          |
| Sui (ed25519)  | `urn:mhda:nt:sui:ci:mainnet:dt:slip10:dp:m/44'/784'/0'/0'/0'`                                        |
| Cardano        | `urn:mhda:nt:cardano:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0`                                  |
| Algorand       | `urn:mhda:nt:algorand:ci:mainnet` (non-HD)                                                           |
| TON            | `urn:mhda:nt:ton:ci:mainnet` (non-HD, friendly base64url default)                                    |
| Cosmos         | `urn:mhda:nt:cosmos:ci:cosmoshub:dt:cip11:dp:m/44'/118'/0'/0/0`                                      |
| EVM short form | `urn:mhda:nt:evm:ci:1` (defaults: secp256k1, hex)                                                    |
| With metadata  | `urn:mhda:nt:evm:ci:1:ct:60` (optional SLIP-44 annotation)                                           |
| Wallet-bound   | `urn:mhda:nt:ton:ci:mainnet:wt:tonconnect:wi:c0a8f2d4-3b6e-4a51-9c7d-2f8e1a0b5c93`                   |

## Specification

Full specification - components, networks, derivation types, formats,
algorithms, validation rules, error sentinels, concurrency model and
known limitations - is in [SPEC.md](./SPEC.md).

## Testing

```sh
# Unit tests + fuzz seed corpora
go test ./...

# Extended fuzzing
go test -run='^$' -fuzz=FuzzParseURN -fuzztime=30s
go test -run='^$' -fuzz=FuzzParseNSS -fuzztime=30s
go test -run='^$' -fuzz=FuzzDerivationPath -fuzztime=30s
```

## License

See [LICENSE](./LICENSE).
