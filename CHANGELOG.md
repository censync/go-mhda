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

### Fixed

- **`ChargeType` is 32 bits wide (was `uint8`).** The CIP-11 charge and the
  CIP-1852 role accept any level index, but the parsed value was truncated
  to a byte: `m/1852'/1815'/0'/256/0` parsed as role 0, so it named the
  role-0 key, re-serialised as `m/1852'/1815'/0'/0/0`, and `Levels()`
  returned the truncated value. The full value is now kept in `Charge()`,
  `Levels()` and `String()`. Code that converts `Charge()` to `uint8` must
  widen its own variable.
