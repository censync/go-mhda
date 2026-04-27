# go-mhda

MultiChain Hierarchical Deterministic Address (MHDA) format - a URN-based
descriptor for blockchain HD addresses, with [RFC 8141](https://datatracker.ietf.org/doc/rfc8141/)
compatibility.

A single string captures everything needed to identify a derived address:
network, derivation scheme, path, signature curve, encoding format and any
prefix/suffix conventions.

```
urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m:ap:bc1p
```

Supported networks: Bitcoin, EVM, Avalanche, Tron, Cosmos, Solana, XRP,
Stellar, NEAR, Aptos, Sui, Cardano, Algorand, TON.

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
    addr, err := mhda.ParseURN("urn:mhda:nt:evm:ct:60:ci:1")
    if err != nil {
        panic(err)
    }
    fmt.Println(addr.Chain().NetworkType()) // evm
    fmt.Println(addr.Algorithm())           // secp256k1 (default for evm)
    fmt.Println(addr.Format())              // hex (default for evm)

    // Strict parsing: also checks the (network, algorithm, format,
    // derivation) combination is in the known-good compatibility matrix.
    if _, err := mhda.ParseURNStrict("urn:mhda:nt:evm:ct:60:ci:1:aa:ed25519"); err != nil {
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

    // The chain-domain triple (nt, ct, ci) is itself a parseable key.
    key := addr.Chain().Key() // "nt:evm:ct:60:ci:1"
    chain, _ := mhda.ChainFromKey(key)
    _ = chain
}
```

## Examples

| Network        | Example URN                                                                                          |
|----------------|------------------------------------------------------------------------------------------------------|
| Ethereum       | `urn:mhda:nt:evm:ct:60:ci:1:dt:bip44:dp:m/44'/60'/0'/0/0`                                            |
| Bitcoin (BIP86)| `urn:mhda:nt:btc:ct:0:ci:bitcoin:dt:bip86:dp:m/86'/0'/0'/0/0:af:bech32m:ap:bc1p`                     |
| Solana         | `urn:mhda:nt:sol:ct:501:ci:mainnet:dt:slip10:dp:m/44'/501'/0'/0'`                                    |
| Stellar        | `urn:mhda:nt:xlm:ct:148:ci:mainnet:dt:slip10:dp:m/44'/148'/0'`                                       |
| Sui (ed25519)  | `urn:mhda:nt:sui:ct:784:ci:mainnet:dt:slip10:dp:m/44'/784'/0'/0'/0'`                                 |
| Cardano        | `urn:mhda:nt:ada:ct:1815:ci:mainnet:dt:cip1852:dp:m/1852'/1815'/0'/0/0`                              |
| Algorand       | `urn:mhda:nt:algo:ct:283:ci:mainnet` (non-HD)                                                        |
| TON            | `urn:mhda:nt:ton:ct:607:ci:mainnet` (non-HD, friendly base64url default)                             |
| Cosmos         | `urn:mhda:nt:cosmos:ct:118:ci:cosmoshub:dt:cip11:dp:m/44'/118'/0'/0/0`                               |
| EVM short form | `urn:mhda:nt:evm:ct:60:ci:1` (defaults: bip44, secp256k1, hex)                                       |

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
