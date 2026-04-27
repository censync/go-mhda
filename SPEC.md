# MHDA Specification

MHDA (MultiChain Hierarchical Deterministic Address) is a URN-based descriptor
for blockchain HD addresses. It is RFC 8141 compatible and unifies the
encoding of derivation paths, signature curves and address formats across
EVM, Bitcoin, Cosmos, Solana, XRP, Stellar, NEAR, Aptos, Sui, Cardano,
Algorand and TON.

This document is the normative specification. The Go reference implementation
in this repository conforms to it.

## 1. URN Structure

### 1.1 Syntax

```
urn:mhda:nt:<network>:ct:<slip44>:ci:<chain_id>:dt:<derivation>:dp:<path>:aa:<algorithm>:af:<format>:ap:<prefix>:as:<suffix>
```

The `urn:` and the NID `mhda` are case-insensitive (RFC 8141 §5.1). Component
keys (`nt`, `dt`, etc.) are lowercase by convention and are accepted only in
that form. The canonical serialised form is fully lowercase.

### 1.2 Components

| Key  | Name                | Required | Type   | Default                  |
|------|---------------------|----------|--------|--------------------------|
| `nt` | Network type        | yes      | string | -                        |
| `ct` | Coin type (SLIP-44) | yes      | uint32 | -                        |
| `ci` | Chain ID            | yes      | string | -                        |
| `dt` | Derivation type     | no       | string | `root`                   |
| `dp` | Derivation path    | no       | string | empty (when `dt=root`)   |
| `aa` | Address algorithm   | no       | string | per-network default      |
| `af` | Address format      | no       | string | per-network default      |
| `ap` | Address prefix      | no       | string | none                     |
| `as` | Address suffix      | no       | string | none                     |

Optional components are emitted in canonical output ONLY when explicitly set.
A short input form round-trips back to the same short form; a long form
round-trips back to the same long form.

### 1.3 Required ordering

Component order in the canonical form is chain-domain first
(`nt`, `ct`, `ci`), then derivation domain (`dt`, `dp`), then optional
address-format metadata (`aa`, `af`, `ap`, `as`). The chain-domain prefix
`nt:<network>:ct:<slip44>:ci:<chain_id>` is itself a valid NSS chain key
returned by `Chain.String()` and consumed by `ChainFromKey` / `ChainFromNSS`.

Parsers MUST accept any ordering on input, and the reference implementation
does so.

### 1.4 RFC 8141 components

Three optional URN structural elements are accepted on input but not
interpreted; they are stripped before parsing:

- `?+<r-component>` (resource request, RFC 8141 §2)
- `?=<q-component>` (query)
- `#<f-component>` (fragment)

Example: `urn:mhda:nt:evm:ct:60:ci:1?+resolver=example.com#sec` parses
identically to the bare URN; `String()` does not preserve these elements.

### 1.5 Charset

NSS values must consist of ASCII characters allowed by RFC 8141 NSS production
(`pchar / "/"` per RFC 3986). The reference implementation does not currently
percent-encode and rejects unescaped whitespace inside values. If a value
needs to contain `:` (currently no in-tree value does), percent-encoding
support must be added.

## 2. Network Catalogue

Every network registers an algorithms set, a formats set, a derivation-types
set, and (optionally) a default algorithm and a default format. Strict
validation (see §6) requires the address' resolved algorithm, format and
derivation type to be members of the corresponding set.

ROOT (no derivation path) is permitted on every network and represents the
non-HD form.

| `nt`   | SLIP-44 | Algorithms                                | Formats                                               | Derivations                            | Default `aa` | Default `af` |
|--------|---------|-------------------------------------------|-------------------------------------------------------|----------------------------------------|--------------|--------------|
| btc    | 0       | secp256k1                                 | p2pkh, p2sh, p2wpkh, p2wsh, p2tr, bech32, bech32m     | bip32, bip44, bip49, bip84, bip86      | secp256k1    | -            |
| evm    | 60      | secp256k1                                 | hex                                                   | bip32, bip44                           | secp256k1    | hex          |
| avm    | 9000    | secp256k1                                 | hex, bech32                                           | bip44                                  | secp256k1    | -            |
| tvm    | 195     | secp256k1                                 | base58                                                | bip44                                  | secp256k1    | base58       |
| cosmos | 118     | secp256k1, ed25519                        | bech32                                                | bip44, cip11                           | secp256k1    | bech32       |
| sol    | 501     | ed25519                                   | base58                                                | slip10                                 | ed25519      | base58       |
| xrp    | 144     | secp256k1, ed25519                        | base58                                                | bip44                                  | secp256k1    | base58       |
| xlm    | 148     | ed25519                                   | strkey                                                | slip10                                 | ed25519      | strkey       |
| near   | 397     | ed25519, secp256k1                        | hex                                                   | slip10, bip44                          | ed25519      | hex          |
| apt    | 637     | ed25519, secp256k1                        | hex                                                   | slip10, bip44                          | ed25519      | hex          |
| sui    | 784     | ed25519, secp256k1, secp256r1             | hex                                                   | slip10, bip54, bip74                   | ed25519      | hex          |
| ada    | 1815    | ed25519                                   | bech32, base58                                        | cip1852                                | ed25519      | bech32       |
| algo   | 283     | ed25519                                   | base32                                                | slip10                                 | ed25519      | base32       |
| ton    | 607     | ed25519                                   | base64url, hex                                        | slip10                                 | ed25519      | base64url    |

### 2.1 Per-network notes

#### Bitcoin (`btc`)
Multiple legitimate scripts; format must be specified explicitly under strict
validation. `ap` is conventionally `1` (P2PKH), `3` (P2SH), `bc1q` (bech32),
`bc1p` (bech32m / Taproot).

#### Ethereum and EVM clones (`evm`)
Single canonical format (hex). Chain ID typically a numeric chain ID such as
`1`, `0xa86a`, `0x10`.

#### Avalanche (`avm`)
C-Chain uses hex (EVM-compatible); X/P-Chain use bech32 with HRPs `X-avax`,
`P-avax`. No default format because both are legitimate.

#### Cosmos (`cosmos`)
Cosmos chains traditionally use BIP-44 with coin type 118; some deployments
register their own SLIP-44 entries. CIP-11 is `m/44'/118'/account'/charge_extra/address`.

#### Solana (`sol`)
SLIP-10 ed25519 only. Common path forms: `m/44'/501'`, `m/44'/501'/account'`,
`m/44'/501'/account'/0'` (Phantom / Solflare convention).

#### XRP Ledger (`xrp`)
secp256k1 is the historical default; ed25519 is supported by newer wallets.
Addresses use base58 with XRPL's custom alphabet (start with `r`).

#### Stellar (`xlm`)
SEP-0005 mandates SLIP-10 ed25519 with `m/44'/148'/account'` (3 levels, all
hardened). StrKey is base32 of `version_byte || payload || CRC16-XMODEM` per
SEP-0023. Common version bytes: `G` (account), `S` (seed), `M` (muxed),
`C` (Soroban contract).

#### NEAR (`near`)
ed25519-implicit accounts use raw 64-char hex of pubkey. ETH-implicit accounts
use `0x` + 40 hex from secp256k1 + keccak256. Named accounts (`alice.near`)
are not HD-derived and not represented by MHDA.

#### Aptos (`apt`)
ed25519 path `m/44'/637'/account'/change'/index'` (all 5 hardened, enforced by
aptos-ts-sdk). secp256k1 path `m/44'/637'/account'/change/index` (BIP-44).

#### Sui (`sui`)
The signature scheme is encoded in the `purpose` field of the path:

```
ed25519:    m/44'/784'/account'/change'/index'   (all 5 hardened, SLIP-10)
secp256k1:  m/54'/784'/account'/change/index     (BIP-32, purpose=54')
secp256r1:  m/74'/784'/account'/change/index     (BIP-32, purpose=74')
```

Address = Blake2b-256(flag || pubkey), 0x + 64 hex chars. Flag bytes:
0x00 ed25519, 0x01 secp256k1, 0x02 secp256r1.

#### Cardano (`ada`)
BIP32-Ed25519 (extended keys with soft-derivation, distinct from SLIP-10
ed25519). Canonical HD path is CIP-1852: `m/1852'/1815'/account'/role/index`.
Roles per CIP-1852: 0=external, 1=internal, 2=staking, 3=DRep,
4=cc-cold, 5=cc-hot. Shelley addresses use bech32 (no BIP-173 length cap, per
CIP-19); Byron-era addresses use base58 and remain on-chain.

#### Algorand (`algo`)
Native scheme is non-HD: a 25-word BIP-39-style mnemonic encodes the
32-byte ed25519 seed directly. Canonical URNs use `dt:root`. Some third-party
wallets layer SLIP-10 at `m/44'/283'/account'/0'/0'`; this is accepted but is
not Algorand-canonical.

#### TON (`ton`)
Native scheme is non-HD: a 24-word TON-specific mnemonic (different word list
from BIP-39) feeds PBKDF2-HMAC-SHA512 to derive a single ed25519 keypair.
Address has two interchangeable forms:

```
raw:           workchain ":" 64 hex chars       (e.g. "0:abcd...")
user-friendly: base64url(tag||workchain||hash||CRC16)  (e.g. "EQCD...")
```

The friendly form is the default user-facing rendering; raw is used inside
protocol messages.

## 3. Derivation Types

| `dt`     | Path template                                         | Levels  | Source                                                                 |
|----------|-------------------------------------------------------|---------|------------------------------------------------------------------------|
| root     | empty                                                 | 0       | -                                                                      |
| bip32    | `m/account'/charge/index[']`                          | 3       | https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki         |
| bip44    | `m/44'/coin'/account'/charge/index[']`                | 5       | https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki         |
| bip49    | `m/49'/coin'/account'/charge/index[']`                | 5       | https://github.com/bitcoin/bips/blob/master/bip-0049.mediawiki         |
| bip54    | `m/54'/coin'/account'/charge/index[']`                | 5       | sui-keys/src/key_derive.rs (Sui secp256k1)                             |
| bip74    | `m/74'/coin'/account'/charge/index[']`                | 5       | sui-keys/src/key_derive.rs (Sui secp256r1)                             |
| bip84    | `m/84'/coin'/account'/charge/index[']`                | 5       | https://github.com/bitcoin/bips/blob/master/bip-0084.mediawiki         |
| bip86    | `m/86'/coin'/account'/charge/index[']`                | 5       | https://github.com/bitcoin/bips/blob/master/bip-0086.mediawiki         |
| slip10   | `m(/uint32['])+`  (variable length)                   | 1+      | https://github.com/satoshilabs/slips/blob/master/slip-0010.md          |
| cip11    | `m/44'/118'/account'/charge_extra/index[']`           | 5       | https://github.com/confio/cosmos-hd-key-derivation-spec                |
| cip1852  | `m/1852'/1815'/account'/role/index[']`                | 5       | https://github.com/cardano-foundation/CIPs/blob/master/CIP-1852/       |
| zip32    | `m/32'/133'/account'[/index[']]`                      | 3 or 4  | https://zips.z.cash/zip-0032                                           |

Hardening markers in input: `'`, `H`, or `h` are all accepted and normalised
to `'` in canonical output. The trailing `[']` on `index` denotes that the
leaf level itself may be hardened.

### 3.1 Variable-length paths

`slip10` accepts any number of levels and is the right type for chains whose
HD derivation does not fit a fixed shape (Solana, Stellar, Aptos ed25519,
Sui ed25519, NEAR, Ledger TON).

`zip32` accepts both 3-level (`m/32'/133'/account'`) and 4-level
(`m/32'/133'/account'/index[']`) forms; both round-trip exactly.

### 3.2 Level inspection

Regardless of derivation type, the canonical level-by-level view is exposed
via `DerivationPath.Levels()`. For BIP-family schemes the BIP-44 shortcut
fields (`Coin`, `Account`, `Charge`, `AddressIndex`) are also populated.

## 4. Algorithms

| `aa`        | Notes                                                                  |
|-------------|------------------------------------------------------------------------|
| secp256k1   | BTC, EVM, Cosmos, Tron, XRP, NEAR (secp256k1 access keys), Aptos, Sui  |
| ed25519     | Solana, Stellar, Cardano, Algorand, TON, NEAR (default), Sui (default) |
| sr25519     | Substrate / Polkadot account keys (not currently registered as a network) |
| secp256r1   | Sui (purpose=74'), some hardware wallets                               |
| secp384r1   | Reserved                                                               |
| secp521r1   | Reserved                                                               |
| prime256v1  | OpenSSL alias for secp256r1; reserved                                  |

## 5. Address Formats

| `af`       | Notes                                                                       |
|------------|-----------------------------------------------------------------------------|
| hex        | 0x-prefixed hex (EVM, Aptos, Sui, NEAR-implicit, TON-raw)                  |
| p2pkh      | Pay to Public Key Hash (legacy BTC, prefix 1)                              |
| p2sh       | Pay to Script Hash (BTC nested SegWit, prefix 3)                           |
| p2wpkh     | Pay to Witness Public Key Hash                                             |
| p2wsh      | Pay to Witness Script Hash                                                 |
| p2tr       | Pay to Taproot (BIP-341)                                                   |
| bech32     | BIP-173 (SegWit v0). Cosmos and Cardano-Shelley also use bech32             |
| bech32m    | BIP-350 (SegWit v1+, used by Taproot)                                      |
| base58     | Bitcoin-alphabet base58check (BTC, XRP custom alphabet, TRX, ADA-Byron)    |
| base32     | RFC 4648 base32 + 4-byte SHA512/256 checksum (Algorand)                    |
| strkey    | Stellar SEP-23: base32 of version_byte || payload || CRC16-XMODEM           |
| base64url  | RFC 4648 base64url, used by TON user-friendly addresses                    |
| ss58       | Substrate/Polkadot SS58 (registered for future use)                        |

## 6. Validation

The library exposes two parsing entry points:

### 6.1 Lenient (`ParseURN`)

Performs structural validation only:
- Prefix `urn:mhda:` (case-insensitive).
- All component values non-empty.
- No duplicate component keys.
- Network type is one of the registered values.
- Coin type is a valid 32-bit unsigned integer (decimal or `0x`-prefixed hex).
- Chain ID is non-empty.
- Derivation type, if present, is one of the registered constants.
- Derivation path, if present, matches the regex of its derivation type.
- Algorithm, if present, is one of the registered constants.
- Format, if present, is one of the registered constants.

It does NOT check that the algorithm/format/derivation are semantically valid
for the given network.

### 6.2 Strict (`ParseURNStrict` / `Address.Validate`)

In addition to lenient checks, strict mode requires that the resolved
`(algorithm, format, derivation type)` triple is in the per-network
compatibility set. Defaults are applied first, so a short URN is validated
as if rewritten in long form.

ROOT is always accepted regardless of network.

### 6.3 Round-trip semantics

The reference implementation guarantees:

- For canonical input: `Parse(s).String() == s`.
- For any successful parse: `Parse(s).String()` re-parses to an equivalent
  representation, idempotent on further round-trips.
- Hardening markers `H`, `h`, `'` are all accepted but always serialised as
  `'`. Inputs using `H` or `h` round-trip to canonical `'` form.
- Whitespace inside NSS values is rejected (RFC 8141 NSS does not permit it).
- Surrounding whitespace around the entire URN is tolerated and trimmed.

## 7. Errors

All errors returned by parsing and validation wrap exported sentinel values.
Use `errors.Is` to discriminate failure modes; do not depend on the message
text.

| Sentinel                     | Trigger                                                  |
|------------------------------|----------------------------------------------------------|
| `ErrInvalidURN`              | Missing or malformed `urn:mhda:` prefix                  |
| `ErrInvalidNSS`              | Malformed namespace-specific string                      |
| `ErrMissingNetworkType`      | `nt` absent                                              |
| `ErrInvalidNetworkType`      | `nt` value not registered                                |
| `ErrMissingCoinType`         | `ct` absent                                              |
| `ErrInvalidCoinType`         | `ct` not a uint32                                        |
| `ErrMissingChainID`          | `ci` absent                                              |
| `ErrInvalidDerivationType`   | `dt` value not registered                                |
| `ErrInvalidDerivationPath`   | `dp` does not match the regex of `dt`                    |
| `ErrInvalidAlgorithm`        | `aa` value not registered                                |
| `ErrInvalidFormat`           | `af` value not registered                                |
| `ErrIncompatible`            | Strict validation: triple not allowed for the network    |
| `ErrUninitializedAddress`    | `MarshalText` / `Validate` called on a zero-value Address |

Error messages typically embed the offending value in quotes for diagnostics
but the textual form is not part of the API contract.

## 8. Concurrency

`Address`, `Chain` and `DerivationPath` are mutable value types; their
`Set*` methods modify the receiver in place and are NOT safe for concurrent
use. Callers must synchronise externally if mutation occurs from multiple
goroutines.

Read-only operations (`String`, `NSS`, all `Hash*`, `MarshalText`,
`Validate`, parser entry points and getter methods) are safe to invoke
concurrently provided the receiver is not being mutated at the same time.

The package-level lookup tables (`networkCompatibility`, `derivationIndex`,
`ntIndex`, `indexAlgorithms`, `indexFormats`, `knownComponents`) are
populated at init time and thereafter read-only; concurrent reads are safe.

## 9. Hashing

Two pairs of hash methods are exposed on `Address`:

| Method        | Algorithm | Output           |
|---------------|-----------|------------------|
| `Hash`        | SHA-1     | hex (40 chars)   |
| `NSSHash`     | SHA-1     | hex (40 chars)   |
| `Hash256`     | SHA-256   | hex (64 chars)   |
| `NSSHash256`  | SHA-256   | hex (64 chars)   |

`Hash` and `NSSHash` are retained for backward compatibility with existing
identifiers. SHA-1 is no longer collision-resistant; new uses should prefer
`Hash256` / `NSSHash256`.

## 10. Chain API

The chain-domain triple `(nt, ct, ci)` is exposed as a standalone `Chain`
type with its own factory functions:

| Constructor                | Purpose                                                  |
|----------------------------|----------------------------------------------------------|
| `NewChain(nt, ct, ci)`     | Programmatic construction.                               |
| `ChainFromNSS(s string)`   | Parse a chain-only NSS form `nt:X:ct:Y:ci:Z`.            |
| `ChainFromKey(key)`        | Parse a `ChainKey` (alias of string) produced by `Key()`.|

`Chain.String()` and `Chain.Key()` both return the canonical chain key
`nt:<network>:ct:<coin>:ci:<chainid>`, suitable for use as a map key, cache
key or content hash input. The format is a strict prefix of any full URN
NSS, so callers can extract a chain key from a `*Address` via
`addr.Chain().Key()`.

## 11. Codec Integration

`Address` implements `encoding.TextMarshaler` and `encoding.TextUnmarshaler`.
This automatically provides:

- `encoding/json` and `encoding/xml` field encoding.
- `gopkg.in/yaml.v3` (and similar third-party YAML encoders).
- Any framework that consults `TextMarshaler` for value serialisation.

`MarshalText` returns the canonical URN form. `UnmarshalText` accepts any
valid input (lenient mode).

A `database/sql` adapter is not currently provided; callers can wrap
`MarshalText`/`UnmarshalText` in their own `driver.Valuer` / `sql.Scanner`.

## 12. Known Limitations

- **Polkadot / Substrate** is not registered as a network type. The native
  derivation scheme uses junctions (`//hard`, `/soft` over BIP-39 entropy)
  rather than BIP-32 paths and does not fit cleanly into the `m/...` path
  model. A future `dt:substrate` could be added if needed.

- **Algorand** and **TON** native schemes are not hierarchical. Their
  canonical URN form uses `dt:root` (no `dp`). HD forms (`m/44'/283'/...` /
  `m/44'/607'/...`) are wallet-specific layering on top, not protocol-canonical.

- **Sui** uses the `purpose` field of the derivation path to encode the
  signature scheme (44'/54'/74'). This is supported via three distinct
  derivation types (`bip44`, `bip54`, `bip74`).

- **Hardening markers** are accepted in three input forms (`'`, `H`, `h`)
  but always serialised canonically as `'`. Round-trip
  `Parse(s).String() == s` therefore holds only for inputs already in
  canonical form.

- **Percent-encoding** is not currently supported. Values may not contain
  `:`. No registered chain currently needs it.

## 13. References

- RFC 8141 - URN Syntax
- RFC 3986 - URI Generic Syntax
- SLIP-0010 - Universal HD wallet derivation
- SLIP-0044 - Coin type registry
- SLIP-0023 - Cardano Ed25519 master node generation
- BIP-0032, BIP-0044, BIP-0049, BIP-0084, BIP-0086 - Bitcoin HD wallets
- BIP-0173, BIP-0350 - bech32 / bech32m
- BIP-0341 - Taproot
- CIP-1852, CIP-19, CIP-5 - Cardano addresses and HD derivation
- SEP-0005, SEP-0023 - Stellar HD derivation and StrKey
- AIP-40 - Aptos canonical address form
