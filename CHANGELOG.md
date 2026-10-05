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

### Fixed

- **`ChargeType` is 32 bits wide (was `uint8`).** The CIP-11 charge and the
  CIP-1852 role accept any level index, but the parsed value was truncated
  to a byte: `m/1852'/1815'/0'/256/0` parsed as role 0, so it named the
  role-0 key, re-serialised as `m/1852'/1815'/0'/0/0`, and `Levels()`
  returned the truncated value. The full value is now kept in `Charge()`,
  `Levels()` and `String()`. Code that converts `Charge()` to `uint8` must
  widen its own variable.
