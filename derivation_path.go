package go_mhda

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	ROOT  = DerivationType(`root`)
	BIP32 = DerivationType(`bip32`)
	BIP44 = DerivationType(`bip44`)
	BIP49 = DerivationType(`bip49`) // P2SH-wrapped SegWit (BIP-49)
	BIP54 = DerivationType(`bip54`) // Sui secp256k1 (purpose=54')
	BIP74 = DerivationType(`bip74`) // Sui secp256r1 (purpose=74')
	BIP84 = DerivationType(`bip84`)
	BIP86 = DerivationType(`bip86`)
	// SLIP10 is the generic SLIP-0010 derivation scheme (ed25519 / curve25519
	// / nist256p1 / secp256k1). Variable-length, per-level hardening; no
	// fixed shape. Used by Solana, Stellar (SEP-0005), Sui-ed25519, Aptos,
	// NEAR, Ledger TON. The leaf-level fields (account/charge/index) are not
	// populated for this type - use Levels() for path inspection.
	SLIP10 = DerivationType(`slip10`)
	// CIP1852 is the Cardano BIP32-Ed25519 scheme, distinct from SLIP-0010
	// because it allows non-hardened soft derivation. Path:
	//   m / 1852' / 1815' / account' / role / index
	// role values per CIP-1852: 0=external, 1=internal, 2=staking, 3=DRep,
	// 4=cc-cold, 5=cc-hot. Stored in the `charge` shortcut field.
	CIP1852 = DerivationType(`cip1852`)
	CIP11   = DerivationType(`cip11`)
	ZIP32   = DerivationType(`zip32`)

	ChargeExternal = ChargeType(0)
	ChargeInternal = ChargeType(1)
)

type DerivationType string

// IsValid reports whether the derivation type is one of the recognised
// constants registered in derivationIndex.
func (dt DerivationType) IsValid() bool {
	_, ok := derivationIndex[dt]
	return ok
}

// String returns the derivation type as a plain string.
func (dt DerivationType) String() string { return string(dt) }

// DerivationTypeFromString parses a string into a DerivationType. The lookup
// is case-insensitive; surrounding whitespace is stripped.
func DerivationTypeFromString(src string) (DerivationType, error) {
	dt := DerivationType(normalize(src))
	if !dt.IsValid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidDerivationType, src)
	}
	return dt, nil
}

type AccountIndex uint32

// ChargeType is the level after the account: the change level of BIP-32 and
// the BIP-44 family (0 external, 1 internal), the CIP-11 charge and the
// CIP-1852 role. It is as wide as a level index: the CIP-11 charge and the
// CIP-1852 role take any index.
type ChargeType uint32

type AddressIndex struct {
	Index      uint32
	IsHardened bool
}

type DerivationPath struct {
	derivationType DerivationType
	coin           CoinType
	account        AccountIndex
	charge         ChargeType
	index          AddressIndex
	hasIndex       bool // true when the leaf address index is part of the path; relevant for variable-length schemes like ZIP-32
	// levels is the canonical, type-agnostic representation of the path.
	// For BIP-family schemes coin/account/charge/index are kept as convenient
	// shortcuts and stay in sync with levels. For schemes with non-BIP shape
	// (SLIP-10 variable length, CIP-1852 with role) levels is the source of
	// truth and the BIP-family shortcuts are left at zero.
	levels []AddressIndex
}

// NewDerivationPath constructs a path for BIP-family schemes (BIP32/44/49/54/
// 74/84/86, CIP-11, CIP-1852, ZIP-32) using the canonical "shortcut" fields.
// For SLIP-10 (variable-length) callers must use NewDerivationPathFromLevels
// instead - SLIP-10 cannot be reconstructed from these five fields and this
// constructor will panic if asked to.
func NewDerivationPath(derivationType DerivationType, coin CoinType, account AccountIndex, charge ChargeType, index AddressIndex) *DerivationPath {
	if derivationType == SLIP10 {
		panic("mhda: NewDerivationPath cannot construct SLIP10 paths; use NewDerivationPathFromLevels")
	}
	dp := &DerivationPath{
		derivationType: derivationType,
		coin:           coin,
		account:        account,
		charge:         charge,
		index:          index,
		hasIndex:       derivationType != ROOT,
	}
	dp.rebuildLevels()
	return dp
}

// NewDerivationPathFromLevels constructs a path from an explicit sequence of
// levels. Required constructor for SLIP-10 (variable length) and useful for
// any scheme when callers prefer the level-array view over BIP-44 shortcuts.
//
// For BIP-family schemes the shortcut fields (coin/account/charge/index) are
// populated from the levels so subsequent String() and getter calls behave the
// same as if the path had been parsed.
func NewDerivationPathFromLevels(derivationType DerivationType, levels []AddressIndex) *DerivationPath {
	cp := make([]AddressIndex, len(levels))
	copy(cp, levels)
	dp := &DerivationPath{
		derivationType: derivationType,
		levels:         cp,
	}
	dp.populateShortcutsFromLevels()
	return dp
}

// populateShortcutsFromLevels fills the BIP-44-style shortcut fields from
// levels[] for derivation types that have a fixed shape, and sets hasIndex to
// reflect whether the leaf-level address index is present. Schemes whose
// layout does not match the shortcut fields (SLIP-10) are left untouched -
// their levels[] is the source of truth.
func (dp *DerivationPath) populateShortcutsFromLevels() {
	switch dp.derivationType {
	case BIP32:
		// m/account'/charge/index
		if len(dp.levels) >= 3 {
			dp.account = AccountIndex(dp.levels[0].Index)
			dp.charge = ChargeType(dp.levels[1].Index)
			dp.index = dp.levels[2]
			dp.hasIndex = true
		}
	case BIP44, BIP49, BIP54, BIP74, BIP84, BIP86, CIP11, CIP1852:
		// m/<purpose>'/<coin>'/account'/change/index - 5 levels
		if len(dp.levels) >= 5 {
			dp.coin = CoinType(dp.levels[1].Index)
			dp.account = AccountIndex(dp.levels[2].Index)
			dp.charge = ChargeType(dp.levels[3].Index)
			dp.index = dp.levels[4]
			dp.hasIndex = true
		}
	case ZIP32:
		// m/32'/133'/account'[/index[']]
		if len(dp.levels) >= 3 {
			dp.coin = CoinType(dp.levels[1].Index)
			dp.account = AccountIndex(dp.levels[2].Index)
		}
		if len(dp.levels) >= 4 {
			dp.index = dp.levels[3]
			dp.hasIndex = true
		}
	case SLIP10:
		dp.hasIndex = len(dp.levels) > 0
	}
}

func ParseDerivationPath(dt DerivationType, path string) (*DerivationPath, error) {
	rx, ok := derivationIndex[dt]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrInvalidDerivationType, dt)
	}

	if !rx.MatchString(path) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidDerivationPath, path)
	}

	dPath := &DerivationPath{derivationType: dt}
	if err := dPath.ParsePath(path); err != nil {
		return nil, err
	}
	return dPath, nil
}

func (dp *DerivationPath) DerivationType() DerivationType {
	return dp.derivationType
}

func (dp *DerivationPath) Coin() CoinType {
	return dp.coin
}

func (dp *DerivationPath) Account() AccountIndex {
	return dp.account
}

func (dp *DerivationPath) Charge() ChargeType {
	return dp.charge
}

func (dp *DerivationPath) AddressIndex() AddressIndex {
	return dp.index
}

func (dp *DerivationPath) IsHardenedAddress() bool {
	return dp.index.IsHardened
}

var (
	rxRoot = regexp.MustCompile(`^$`)

	// https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
	// m / account ' / charge / address
	rxBip32 = regexp.MustCompile(`^m/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki
	// m / 44 ' / coin ' / account ' / charge / address
	rxBip44 = regexp.MustCompile(`^m/44[Hh']/([0-9]+)[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/bitcoin/bips/blob/master/bip-0049.mediawiki
	// m / 49 ' / coin ' / account ' / charge / address  (P2SH-wrapped SegWit)
	rxBip49 = regexp.MustCompile(`^m/49[Hh']/([0-9]+)[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/MystenLabs/sui/blob/main/crates/sui-keys/src/key_derive.rs
	// Sui secp256k1: m / 54 ' / coin ' / account ' / change / address
	rxBip54 = regexp.MustCompile(`^m/54[Hh']/([0-9]+)[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/MystenLabs/sui/blob/main/crates/sui-keys/src/key_derive.rs
	// Sui secp256r1: m / 74 ' / coin ' / account ' / change / address
	rxBip74 = regexp.MustCompile(`^m/74[Hh']/([0-9]+)[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/bitcoin/bips/blob/master/bip-0084.mediawiki
	// m / 84 ' / coin ' / account ' / charge / address
	rxBip84 = regexp.MustCompile(`^m/84[Hh']/([0-9]+)[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/bitcoin/bips/blob/master/bip-0086.mediawiki
	// m / 86 ' / coin ' / account ' / charge / address
	rxBip86 = regexp.MustCompile(`^m/86[Hh']/([0-9]+)[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/satoshilabs/slips/blob/master/slip-0010.md
	// Generic SLIP-0010: any number of levels, each optionally hardened.
	// Per-level structure is parsed manually since the regex only validates
	// shape, not per-level extraction.
	rxSlip10 = regexp.MustCompile(`^m(?:/[0-9]+[Hh']?)+$`)

	// https://github.com/cardano-foundation/CIPs/blob/master/CIP-1852/README.md
	// m / 1852 ' / 1815 ' / account ' / role / index
	// role accepts any non-negative integer; CIP-1852 currently defines 0..5
	// but stays liberal here so future CIP role assignments don't reject.
	rxCip1852 = regexp.MustCompile(`^m/1852[Hh']/1815[Hh']/([0-9]+)[Hh']/([0-9]+)/([0-9]+)([Hh'])?$`)

	// https://github.com/confio/cosmos-hd-key-derivation-spec
	// m / 44 ' / 118 ' / account ' / charge_extra / address
	rxCip11 = regexp.MustCompile(`^m/44[Hh']/118[Hh']/([0-9]+)[Hh']/([0-9]+)/([0-9]+)([Hh'])?$`)

	// https://zips.z.cash/zip-0032
	// m / 32 ' / 133 ' / account '
	// m / 32 ' / 133 ' / account ' / address
	// m / 32 ' / 133 ' / account ' / address '
	rxZip32 = regexp.MustCompile(`^m/32[Hh']/133[Hh']/([0-9]+)[Hh'](?:/([0-9]+)([Hh'])?)?$`)

	derivationIndex = map[DerivationType]*regexp.Regexp{
		ROOT:    rxRoot,
		BIP32:   rxBip32,
		BIP44:   rxBip44,
		BIP49:   rxBip49,
		BIP54:   rxBip54,
		BIP74:   rxBip74,
		BIP84:   rxBip84,
		BIP86:   rxBip86,
		SLIP10:  rxSlip10,
		CIP1852: rxCip1852,
		CIP11:   rxCip11,
		ZIP32:   rxZip32,
	}
)

func (dp *DerivationPath) ParsePath(path string) error {
	rx, ok := derivationIndex[dp.derivationType]
	if !ok {
		return fmt.Errorf("%w: %q", ErrInvalidDerivationType, dp.derivationType)
	}

	if dp.derivationType == ROOT {
		if path != "" {
			return fmt.Errorf("%w: root derivation must have empty path, got %q", ErrInvalidDerivationPath, path)
		}
		return nil
	}

	matches := rx.FindStringSubmatch(path)
	if matches == nil {
		return fmt.Errorf("%w: %q", ErrInvalidDerivationPath, path)
	}

	parseUint := func(s, label string) (uint32, error) {
		v, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("%w: cannot parse %s %q: %s", ErrInvalidDerivationPath, label, s, err)
		}
		return uint32(v), nil
	}

	switch dp.derivationType {
	case BIP32:
		// matches: [_, account, charge, index, hardenedMark]
		account, err := parseUint(matches[1], "account")
		if err != nil {
			return err
		}
		charge, err := parseUint(matches[2], "charge")
		if err != nil {
			return err
		}
		index, err := parseUint(matches[3], "index")
		if err != nil {
			return err
		}
		dp.account = AccountIndex(account)
		dp.charge = ChargeType(charge)
		dp.index = AddressIndex{Index: index, IsHardened: matches[4] != ""}
		dp.hasIndex = true

	case BIP44, BIP49, BIP54, BIP74, BIP84, BIP86:
		// matches: [_, coin, account, charge, index, hardenedMark]
		coin, err := parseUint(matches[1], "coin")
		if err != nil {
			return err
		}
		account, err := parseUint(matches[2], "account")
		if err != nil {
			return err
		}
		charge, err := parseUint(matches[3], "charge")
		if err != nil {
			return err
		}
		index, err := parseUint(matches[4], "index")
		if err != nil {
			return err
		}
		dp.coin = CoinType(coin)
		dp.account = AccountIndex(account)
		dp.charge = ChargeType(charge)
		dp.index = AddressIndex{Index: index, IsHardened: matches[5] != ""}
		dp.hasIndex = true

	case CIP1852:
		// matches: [_, account, role, index, hardenedMark]
		// purpose is fixed (1852') and coin is fixed (1815') by the spec
		account, err := parseUint(matches[1], "account")
		if err != nil {
			return err
		}
		role, err := parseUint(matches[2], "role")
		if err != nil {
			return err
		}
		index, err := parseUint(matches[3], "index")
		if err != nil {
			return err
		}
		dp.coin = CoinType(1815)
		dp.account = AccountIndex(account)
		dp.charge = ChargeType(role)
		dp.index = AddressIndex{Index: index, IsHardened: matches[4] != ""}
		dp.hasIndex = true

	case CIP11:
		// matches: [_, account, charge, index, hardenedMark]
		// coin is fixed (118) by the spec
		account, err := parseUint(matches[1], "account")
		if err != nil {
			return err
		}
		charge, err := parseUint(matches[2], "charge")
		if err != nil {
			return err
		}
		index, err := parseUint(matches[3], "index")
		if err != nil {
			return err
		}
		dp.coin = CoinType(118)
		dp.account = AccountIndex(account)
		dp.charge = ChargeType(charge)
		dp.index = AddressIndex{Index: index, IsHardened: matches[4] != ""}
		dp.hasIndex = true

	case SLIP10:
		// Generic SLIP-0010: split the path manually to extract per-level
		// index and hardening marker. The regex above only validates shape.
		segments := strings.Split(path[2:], "/") // skip leading "m/"
		levels := make([]AddressIndex, 0, len(segments))
		for i, seg := range segments {
			hardened := false
			if n := len(seg); n > 0 {
				switch seg[n-1] {
				case '\'', 'h', 'H':
					hardened = true
					seg = seg[:n-1]
				}
			}
			v, err := parseUint(seg, fmt.Sprintf("level %d", i))
			if err != nil {
				return err
			}
			levels = append(levels, AddressIndex{Index: v, IsHardened: hardened})
		}
		dp.levels = levels
		// SLIP-10 paths do not map cleanly onto the BIP-44 shortcut fields
		// (account/charge/index) - leave them at zero. Consumers should use
		// Levels() to inspect the path.
		dp.hasIndex = true

	case ZIP32:
		// matches: [_, account, index?, hardenedMark?]
		// coin is fixed (133) by the spec
		account, err := parseUint(matches[1], "account")
		if err != nil {
			return err
		}
		dp.coin = CoinType(133)
		dp.account = AccountIndex(account)
		if matches[2] != "" {
			index, err := parseUint(matches[2], "index")
			if err != nil {
				return err
			}
			dp.index = AddressIndex{Index: index, IsHardened: matches[3] != ""}
			dp.hasIndex = true
		}

	default:
		return fmt.Errorf("%w: %q", ErrInvalidDerivationType, dp.derivationType)
	}

	dp.rebuildLevels()
	return nil
}

// fixedPrefix returns the (purpose, coin) pair that a derivation type pins
// before account/change/index levels. Both values are hardened. ok=false for
// types that have no fixed prefix (BIP32, ROOT, SLIP10).
func (dp *DerivationPath) fixedPrefix() (purpose, coin uint32, ok bool) {
	switch dp.derivationType {
	case BIP44:
		return 44, uint32(dp.coin), true
	case BIP49:
		return 49, uint32(dp.coin), true
	case BIP54:
		return 54, uint32(dp.coin), true
	case BIP74:
		return 74, uint32(dp.coin), true
	case BIP84:
		return 84, uint32(dp.coin), true
	case BIP86:
		return 86, uint32(dp.coin), true
	case CIP11:
		return 44, 118, true
	case CIP1852:
		return 1852, 1815, true
	case ZIP32:
		return 32, 133, true
	}
	return 0, 0, false
}

// rebuildLevels populates the canonical levels[] from the BIP-family shortcut
// fields based on the derivation type. Called by ParsePath and
// NewDerivationPath so that Levels() returns a correct view regardless of
// which constructor was used.
func (dp *DerivationPath) rebuildLevels() {
	switch dp.derivationType {
	case ROOT:
		dp.levels = nil
		return
	case SLIP10:
		// SLIP-10 levels[] is the source of truth; populated directly by
		// ParsePath / NewDerivationPathFromLevels.
		return
	case BIP32:
		dp.levels = []AddressIndex{
			{Index: uint32(dp.account), IsHardened: true},
			{Index: uint32(dp.charge), IsHardened: false},
			dp.index,
		}
		return
	}

	purpose, coin, ok := dp.fixedPrefix()
	if !ok {
		return
	}

	dp.levels = []AddressIndex{
		{Index: purpose, IsHardened: true},
		{Index: coin, IsHardened: true},
		{Index: uint32(dp.account), IsHardened: true},
	}

	if dp.derivationType == ZIP32 {
		// ZIP-32 has 3 or 4 levels; charge is not part of the spec.
		if dp.hasIndex {
			dp.levels = append(dp.levels, dp.index)
		}
		return
	}

	dp.levels = append(dp.levels,
		AddressIndex{Index: uint32(dp.charge), IsHardened: false},
		dp.index,
	)
}

// Levels returns the canonical level-by-level view of the derivation path.
// Empty for ROOT.
func (dp *DerivationPath) Levels() []AddressIndex {
	return dp.levels
}

// String returns the canonical textual form of the derivation path.
// Hardened markers are emitted as `'` regardless of which marker (`'`, `H`,
// `h`) appeared in the input.
func (dp *DerivationPath) String() string {
	switch dp.derivationType {
	case ROOT:
		return ``
	case SLIP10:
		return formatLevels(dp.levels)
	case BIP32:
		// m/account'/charge/index
		out := fmt.Sprintf("m/%d'/%d/%d", dp.account, dp.charge, dp.index.Index)
		if dp.index.IsHardened {
			out += `'`
		}
		return out
	}

	purpose, coin, ok := dp.fixedPrefix()
	if !ok {
		return ``
	}

	if dp.derivationType == ZIP32 {
		// ZIP-32: m/32'/133'/account'[/index[']]  (no charge)
		base := fmt.Sprintf("m/%d'/%d'/%d'", purpose, coin, dp.account)
		if !dp.hasIndex {
			return base
		}
		base += fmt.Sprintf("/%d", dp.index.Index)
		if dp.index.IsHardened {
			base += `'`
		}
		return base
	}

	// BIP-44 family + CIP-11/CIP-1852: m/<purpose>'/<coin>'/account'/charge/index
	out := fmt.Sprintf("m/%d'/%d'/%d'/%d/%d", purpose, coin, dp.account, dp.charge, dp.index.Index)
	if dp.index.IsHardened {
		out += `'`
	}
	return out
}

// formatLevels renders an arbitrary level slice as `m/<i0>[']/<i1>[']/...`.
// Used for SLIP-10 paths where the path shape is variable.
func formatLevels(levels []AddressIndex) string {
	var b strings.Builder
	b.WriteString("m")
	for _, lvl := range levels {
		fmt.Fprintf(&b, "/%d", lvl.Index)
		if lvl.IsHardened {
			b.WriteString("'")
		}
	}
	return b.String()
}
