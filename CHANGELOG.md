# Changelog

All notable changes to this project will be documented here. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Releases before 1.2.0 are described by their tags and commit messages.

## [1.2.0] — 2026-10-05

### Fixed

- **`ChargeType` is 32 bits wide (was `uint8`).** The CIP-11 charge and the
  CIP-1852 role accept any level index, but the parsed value was truncated
  to a byte: `m/1852'/1815'/0'/256/0` parsed as role 0, so it named the
  role-0 key, re-serialised as `m/1852'/1815'/0'/0/0`, and `Levels()`
  returned the truncated value. The full value is now kept in `Charge()`,
  `Levels()` and `String()`. Code that converts `Charge()` to `uint8` must
  widen its own variable.
