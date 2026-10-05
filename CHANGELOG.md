# Changelog

All notable changes to this project will be documented here. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Releases before 1.2.0 are described by their tags and commit messages.

## [1.2.0] — 2026-10-05

URNs accepted by 1.1 are refused now: a derivation path with a level index
of 2^31 or more no longer parses.

### Changed

- **A derivation-path level index of 2^31 or more is refused** with
  `ErrInvalidDerivationPath`, at every level of every derivation type
  (`bip32`, `bip44`, `bip49`, `bip54`, `bip74`, `bip84`, `bip86`, `cip11`,
  `cip1852`, `zip32`, `slip10`), hardened or not. 1.1 accepted any value up
  to 2^32-1. A BIP-32 child number keeps the hardened flag in its top bit
  (`n'` is child number 2^31+n), so such an index aliased another key: a
  consumer computing `hardened ? 0x80000000 | index : index` derives the key
  of `m/44'/60'/0'/0/0` for `m/44'/60'/2147483648'/0/0`, and an unhardened
  `m/44'/60'/0'/0/2147483648` asks for a hardened child number through the
  public-key formula. No longer parse: any `dp` with a level of
  `2147483648` to `4294967295`, with or without a hardened marker, e.g.
  `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/2147483648'/0/0` and
  `urn:mhda:nt:evm:ci:1:dt:bip44:dp:m/44'/60'/0'/0/4294967295`. The largest
  index is `2147483647`. The `ct` component is not a path level and keeps
  its 32-bit range.
- Leading zeros in a path level are unchanged and now documented: accepted
  in a variable level and dropped in canonical output (`060'` is `60'`),
  refused in a fixed level (the purpose, the fixed coin of `cip11`,
  `cip1852` and `zip32`, the `0`/`1` charge of `bip32` and the BIP-44
  family).
- **Programmatic values are validated like parsed input.** The network
  type, the chain id and the derivation type are written verbatim into
  every URN, and nothing checked them when set in code: a chain id
  `1:dt:bip44:dp:m/44'/60'/0'/0/666` on a root address produced a URN that
  re-parsed as a bip44 path. `NewChain` now requires a registered network
  type and a non-empty chain id of printable ASCII without `:`, `?` or `#`
  (ASCII-trimmed) and panics otherwise, like `NewAddress`.
  `Chain.SetNetworkType` and `Chain.SetChainId` return an error
  (`ErrInvalidNetworkType`, `ErrMissingChainID`, `ErrInvalidValue`) and
  leave the chain unchanged; they returned nothing before.
  `NewDerivationPath` and `NewDerivationPathFromLevels` panic with
  `ErrInvalidDerivationType` on an unregistered type. An `Address` whose
  path has no type serialises like a root address instead of emitting
  empty `dt`/`dp` components.
- **A derivation path without a derivation type, or under `dt:root`, is
  refused** with `ErrInvalidDerivationPath`. Both used to be dropped
  silently: `urn:mhda:nt:evm:ci:1:dp:m/44'/60'/0'/0/0` and
  `urn:mhda:nt:evm:ci:1:dt:root:dp:m/0` parsed as the root address
  `urn:mhda:nt:evm:ci:1`, naming the root key instead of the path they
  spell out. `Address.SetDerivationPath` on a root address accepts only an
  empty path.
- **The NSS is parsed strictly as `key:value` pairs.** An unknown key was
  skipped as a single token, so its value was read as the next key and a
  key with the wrong case vanished: `...:ci:1:DT:bip44:DP:m/44'/60'/0'/0/5`
  parsed as the root address `urn:mhda:nt:evm:ci:1`, and
  `...:ci:1:ext:wi:dt:bip44:...` read `dt` as the wallet id. An unknown
  component is now skipped together with its value; a key that differs from
  a known key only by case, a dangling token, a trailing `:` and an empty
  key are refused with `ErrInvalidNSS`. `ChainFromKey` reports a dangling
  token as `ErrInvalidChainKey`, as before.
- **`ParseNSS`, `ChainFromNSS` and `ChainFromKey` refuse `?` and `#`** with
  `ErrInvalidNSS`. `ParseURN` strips the RFC 8141 r/q/f components before
  parsing, but an NSS given on its own kept the byte in a value:
  `ParseNSS("nt:evm:ci:1?=q:dt:bip44:dp:m/44'/60'/0'/0/0")` returned a bip44
  address whose URN re-parsed as the root address `urn:mhda:nt:evm:ci:1`,
  and `ChainFromKey("nt:evm:ci:1#a")` was accepted. `FuzzParseNSS` now
  checks that the emitted URN re-parses to itself.
- **A `slip10` path has at most 255 levels**; a deeper one is refused with
  `ErrInvalidDerivationPath`. BIP-32 serialises a key's depth in one byte,
  so no wallet can represent a deeper key.

### Documentation

- SPEC.md states that an explicit `dt:root` is folded away (root is the
  default and has no path, so a root address has one canonical form and
  one hash); the Algorand and TON notes no longer call `dt:root` the
  canonical form. It also lists ZIP-32 as a known limitation: `zip32`
  parses, but no network registers it, so strict parsing refuses it. Both
  behaviours are unchanged and now pinned by tests.

- **Strict validation refuses curve, purpose and format combinations no
  wallet can derive.** It checked the algorithm, the format and the
  derivation type each on its own, so it accepted SLIP-10 ed25519 paths
  with soft levels (`nt:solana:...:dp:m/44'/501'/0/0`; ed25519 has no soft
  derivation), `bip44` with ed25519 on XRPL, NEAR and Aptos, Sui `bip54`
  with ed25519 or secp256r1, a Bitcoin `bip84` path with `af:p2pkh`, and
  Cosmos `bip44` with coin `118'`, which is the `cip11` path again.
  `ParseURNStrict` / `Validate` now return `ErrIncompatible` for: an
  ed25519 path with an unhardened level (except `cip1852`, which is
  BIP32-Ed25519); a derivation type with another curve than its own on Sui
  (`slip10` ed25519, `bip54` secp256k1, `bip74` secp256r1), Aptos and NEAR
  (`slip10` ed25519, `bip44` secp256k1); an explicit Bitcoin format that
  does not match the purpose (`bip44` p2pkh, `bip49` p2sh, `bip84` p2wpkh
  or bech32, `bip86` p2tr or bech32m); and Cosmos `bip44` with coin
  `118'`. A Bitcoin URN without `af` stays valid; SPEC.md no longer claims
  strict mode requires it. Lenient parsing is unchanged.

### Fixed

- **No nil dereference on a zero or partial `Address`.** `String()`,
  `NSS()`, the hashes, `Algorithm()` and `Format()` panicked on the zero
  `Address` or one built with a nil chain, and `SetDerivationPath` panicked
  on an address built without a path. They now treat a missing chain as the
  zero chain (`urn:mhda:nt::ci:`, as the C++ port's default address) and a
  missing path as root; `SetCoinType` without a chain returns
  `ErrUninitializedAddress`.
- **An `Address` held by value encodes as its URN.** `MarshalText` had a
  pointer receiver, so an `Address` struct field or map value was encoded
  by `encoding/json` and `encoding/xml` as an empty object (`{"A":{}}`),
  which then failed to decode. `MarshalText` now has a value receiver. A
  nil `*Address` is still encoded by the codec (as `null`); calling
  `MarshalText` directly on a nil pointer now panics like any value method
  instead of returning `ErrUninitializedAddress`.
- **No stale or mixed derivation state.** `ParsePath` updated the fields
  of the path it was given: re-parsing a 4-level ZIP-32 path with a 3-level
  one kept the old index (`m/32'/133'/1'` read back as `m/32'/133'/1'/5`),
  a SLIP-10 path kept the coin, account and charge of a BIP-44 path parsed
  before it, and an error midway left a half-updated path.
  `SetDerivationType` changed the type and kept the old path, so a bip44
  address switched to zip32 serialised as `dt:zip32:dp:m/32'/133'/3'/9`, a
  valid URN naming a key nobody gave. `ParsePath` now replaces the whole
  path or nothing. A new derivation type drops the old path; until a path
  is set the URN carries `dt` without `dp` (and does not parse), and
  `Validate` / `MarshalText` return `ErrInvalidDerivationPath`. The new
  `Address.SetDerivation(dt, dp)` sets both at once and changes nothing on
  error; the URN parser uses it.
- **`Levels()` and `String()` cannot disagree.** `NewDerivationPathFromLevels`
  kept any levels it was given while `String()` printed the type's
  template: a BIP-44 path built from `49'/60'/0/0/0` printed
  `m/44'/60'/0'/0/0` but returned purpose 49' and an unhardened account from
  `Levels()`, and too few levels or an index of 2^31 produced a URN that
  does not parse. Both constructors now return exactly the path the parser
  gives back for `String()`, and panic with `ErrInvalidDerivationPath`
  otherwise. `Levels()` returned the internal slice, so a caller that
  changed it changed the path; it returns a copy. `NewAddress` stored the
  caller's chain and path, so two addresses built from them changed
  together; it keeps copies.
- **`ChargeType` is 32 bits wide (was `uint8`).** The CIP-11 charge and the
  CIP-1852 role accept any level index, but the parsed value was truncated
  to a byte: `m/1852'/1815'/0'/256/0` parsed as role 0, so it named the
  role-0 key, re-serialised as `m/1852'/1815'/0'/0/0`, and `Levels()`
  returned the truncated value. The full value is now kept in `Charge()`,
  `Levels()` and `String()`. Code that converts `Charge()` to `uint8` must
  widen its own variable.
