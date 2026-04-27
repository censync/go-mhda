package go_mhda

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type MHDA interface {
	Chain() *Chain
	DerivationType() DerivationType
	DerivationPath() *DerivationPath
	Algorithm() Algorithm
	Format() Format
	NSS() string
	String() string
	// Hash returns a SHA-1 digest of String(). Retained for backward
	// compatibility; SHA-1 is no longer collision-resistant. Prefer Hash256.
	Hash() string
	// NSSHash returns a SHA-1 digest of NSS(). Same caveat as Hash.
	NSSHash() string
	// Hash256 returns the SHA-256 digest of String(), hex-encoded.
	Hash256() string
	// NSSHash256 returns the SHA-256 digest of NSS(), hex-encoded.
	NSSHash256() string
}

type Address struct {
	chain            *Chain
	path             *DerivationPath
	addressAlgorithm Algorithm
	addressFormat    Format
	addressPrefix    string
	addressSuffix    string
}

// NewAddress builds an Address from chain and derivation path. Optional params,
// in order, are: addressAlgorithm (aa), addressFormat (af), addressPrefix (ap),
// addressSuffix (as). Empty strings are treated as unset.
func NewAddress(chain *Chain, path *DerivationPath, params ...string) *Address {
	a := &Address{chain: chain, path: path}
	get := func(i int) string {
		if i < len(params) {
			return params[i]
		}
		return ""
	}
	if v := get(0); v != "" {
		a.addressAlgorithm = Algorithm(normalize(v))
	}
	if v := get(1); v != "" {
		a.addressFormat = Format(strings.TrimSpace(v))
	}
	if v := get(2); v != "" {
		a.addressPrefix = strings.TrimSpace(v)
	}
	if v := get(3); v != "" {
		a.addressSuffix = strings.TrimSpace(v)
	}
	return a
}

func parseAddress(m map[string]string) (MHDA, error) {
	chain, err := parseChain(m)
	if err != nil {
		return nil, err
	}

	mhda := &Address{chain: chain}

	if err := mhda.SetDerivationType(m[compDerivationType]); err != nil {
		return nil, err
	}
	if err := mhda.SetDerivationPath(m[compDerivationPath]); err != nil {
		return nil, err
	}
	if err := mhda.SetAddressAlgorithm(m[compAddressAlgorithm]); err != nil {
		return nil, err
	}
	if err := mhda.SetAddressFormat(m[compAddressFormat]); err != nil {
		return nil, err
	}
	if err := mhda.SetAddressPrefix(m[compAddressPrefix]); err != nil {
		return nil, err
	}
	if err := mhda.SetAddressSuffix(m[compAddressSuffix]); err != nil {
		return nil, err
	}
	return mhda, nil
}

func (a *Address) Chain() *Chain {
	return a.chain
}

// DerivationType returns the derivation type of the address' path, or ROOT
// if the address has no path.
func (a *Address) DerivationType() DerivationType {
	if a.path == nil {
		return ROOT
	}
	return a.path.derivationType
}

func (a *Address) DerivationPath() *DerivationPath {
	return a.path
}

// Algorithm returns the explicitly set algorithm or, if none was set, the
// default for the address' network type (see compatibility.go).
func (a *Address) Algorithm() Algorithm {
	if a.addressAlgorithm != "" {
		return a.addressAlgorithm
	}
	return defaultAlgorithm(a.chain.networkType)
}

// Format returns the explicitly set address format or, if none was set, the
// default for the address' network type. Networks with multiple legitimate
// formats (e.g. Bitcoin) return "" when nothing was set explicitly.
func (a *Address) Format() Format {
	if a.addressFormat != "" {
		return a.addressFormat
	}
	return defaultFormat(a.chain.networkType)
}

func (a *Address) SetDerivationType(dt string) error {
	dt = strings.TrimSpace(dt)
	dt = strings.ToLower(dt)

	if a.path == nil {
		a.path = &DerivationPath{}
	}

	if dt != `` {
		if _, ok := derivationIndex[DerivationType(dt)]; !ok {
			return fmt.Errorf("%w: %q", ErrInvalidDerivationType, dt)
		}

		a.path.derivationType = DerivationType(dt)
	} else {
		a.path.derivationType = ROOT
	}

	return nil
}

func (a *Address) SetDerivationPath(dp string) error {
	if a.path.derivationType == ROOT {
		return nil
	}

	rx, ok := derivationIndex[a.path.derivationType]
	if !ok {
		return fmt.Errorf("%w: unknown derivation type %q", ErrInvalidDerivationPath, a.path.derivationType)
	}

	dp = strings.TrimSpace(dp)
	dp = strings.ToLower(dp)

	if !rx.MatchString(dp) {
		return fmt.Errorf("%w: %q", ErrInvalidDerivationPath, dp)
	}

	// ParsePath wraps its own errors with the appropriate sentinel; surface
	// the result unchanged so callers can errors.Is(...) the inner sentinel.
	return a.path.ParsePath(dp)
}

func (a *Address) SetCoinType(ct string) error {
	ct = strings.TrimSpace(ct)
	if ct == `` {
		return ErrMissingCoinType
	}

	coinType, err := strconv.ParseUint(ct, 0, 32)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidCoinType, ct)
	}

	a.chain.coinType = CoinType(coinType)
	return nil
}

func (a *Address) SetAddressAlgorithm(aa string) error {
	aa = strings.TrimSpace(aa)
	aa = strings.ToLower(aa)
	if aa == `` {
		a.addressAlgorithm = ""
		return nil
	}
	if _, ok := indexAlgorithms[Algorithm(aa)]; !ok {
		return fmt.Errorf("%w: %q", ErrInvalidAlgorithm, aa)
	}
	a.addressAlgorithm = Algorithm(aa)
	return nil
}

func (a *Address) SetAddressFormat(af string) error {
	af = strings.TrimSpace(af)
	af = strings.ToLower(af)
	if af == `` {
		a.addressFormat = ""
		return nil
	}
	if _, ok := indexFormats[Format(af)]; !ok {
		return fmt.Errorf("%w: %q", ErrInvalidFormat, af)
	}
	a.addressFormat = Format(af)
	return nil
}

// SetAddressPrefix sets the optional address prefix. Passing an empty string
// resets the prefix - matching the semantics of SetAddressAlgorithm and
// SetAddressFormat.
func (a *Address) SetAddressPrefix(ap string) error {
	a.addressPrefix = strings.TrimSpace(ap)
	return nil
}

// SetAddressSuffix sets the optional address suffix. Passing an empty string
// resets the suffix.
func (a *Address) SetAddressSuffix(as string) error {
	a.addressSuffix = strings.TrimSpace(as)
	return nil
}

func (a *Address) String() string {
	return fmt.Sprintf(`urn:mhda:%s`, a.NSS())
}

// NSS returns the URN namespace-specific string in canonical form. The
// emission order is the chain-domain (nt/ct/ci) first, then the optional
// derivation domain (dt/dp), then optional address-format metadata
// (aa/af/ap/as). Optional components are emitted only when explicitly set,
// preserving the round-trip with short input forms.
func (a *Address) NSS() string {
	var b strings.Builder

	// Chain domain - always present.
	fmt.Fprintf(&b, "nt:%s:ct:%d:ci:%s", a.chain.networkType, a.chain.coinType, a.chain.chainId)

	// Derivation domain - present when not ROOT.
	if a.path != nil && a.path.derivationType != ROOT {
		b.WriteString(":dt:")
		b.WriteString(string(a.path.derivationType))
		b.WriteString(":dp:")
		b.WriteString(a.path.String())
	}

	// Address-format metadata - emitted only when explicitly set.
	if a.addressAlgorithm != "" {
		b.WriteString(":aa:")
		b.WriteString(string(a.addressAlgorithm))
	}
	if a.addressFormat != "" {
		b.WriteString(":af:")
		b.WriteString(string(a.addressFormat))
	}
	if a.addressPrefix != "" {
		b.WriteString(":ap:")
		b.WriteString(a.addressPrefix)
	}
	if a.addressSuffix != "" {
		b.WriteString(":as:")
		b.WriteString(a.addressSuffix)
	}

	return b.String()
}

// Hash returns a SHA-1 digest of String() as hex. Retained for backward
// compatibility with existing identifiers; SHA-1 is broken under collision
// attacks and should not be relied upon for new uses. Prefer Hash256.
func (a *Address) Hash() string {
	h := sha1.New()
	h.Write([]byte(a.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// NSSHash returns a SHA-1 digest of NSS() as hex. Same caveat as Hash.
func (a *Address) NSSHash() string {
	h := sha1.New()
	h.Write([]byte(a.NSS()))
	return hex.EncodeToString(h.Sum(nil))
}

// Hash256 returns a SHA-256 digest of String() as hex. Use this for content-
// addressing or deduplication keys.
func (a *Address) Hash256() string {
	sum := sha256.Sum256([]byte(a.String()))
	return hex.EncodeToString(sum[:])
}

// NSSHash256 returns a SHA-256 digest of NSS() as hex.
func (a *Address) NSSHash256() string {
	sum := sha256.Sum256([]byte(a.NSS()))
	return hex.EncodeToString(sum[:])
}

// MarshalText implements encoding.TextMarshaler. This is the integration point
// for encoding/json, encoding/xml, gopkg.in/yaml.v3 and similar codecs - they
// will produce the URN form automatically.
func (a *Address) MarshalText() ([]byte, error) {
	if a == nil || a.chain == nil {
		return nil, ErrUninitializedAddress
	}
	return []byte(a.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (a *Address) UnmarshalText(data []byte) error {
	parsed, err := ParseURN(string(data))
	if err != nil {
		return err
	}
	p, ok := parsed.(*Address)
	if !ok {
		return ErrInvalidURN
	}
	*a = *p
	return nil
}
