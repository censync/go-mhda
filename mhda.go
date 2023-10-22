package go_mhda

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type MHDA interface {
	Chain() Chain
	// DerivationType() DerivationType
	DerivationPath() *DerivationPath
	Algorithm() Algorithm
	Format() Format
	NSS() string
	String() string
	Hash() string
	NSSHash() string
}

type Address struct {
	chain            *Chain
	path             *DerivationPath
	addressAlgorithm Algorithm
	addressFormat    Format
	addressPrefix    string
	addressSuffix    string
}

// NewAddress  add optional params: aa, af, ap, as
func NewAddress(chain *Chain, path *DerivationPath, params ...string) *Address {
	return &Address{chain: chain, path: path}
}

func parseAddress(m map[string]string) (MHDA, error) {
	var err error

	chain, err := parseChain(m)

	if err != nil {
		return nil, err
	}

	mhda := &Address{
		chain: chain,
	}

	err = mhda.SetDerivationType(m[compDerivationType])
	if err != nil {
		return nil, err
	}

	// TODO: Add dp validation
	err = mhda.SetDerivationPath(m[compDerivationPath])
	if err != nil {
		return nil, err
	}

	err = mhda.SetAddressAlgorithm(m[compAddressAlgorithm])
	if err != nil {
		return nil, err
	}

	err = mhda.SetAddressFormat(m[compAddressFormat])
	if err != nil {
		return nil, err
	}

	err = mhda.SetAddressPrefix(m[compAddressPrefix])
	if err != nil {
		return nil, err
	}

	err = mhda.SetAddressSuffix(m[compAddressSuffix])
	if err != nil {
		return nil, err
	}

	return mhda, nil
}

func (a *Address) Chain() Chain {
	return *a.chain
}

/*func (a *Address) DerivationType() DerivationType {
	return a.path.derivationType
}*/

func (a *Address) DerivationPath() *DerivationPath {
	return a.path
}

func (a *Address) Algorithm() Algorithm {
	return a.addressAlgorithm
}

func (a *Address) Format() Format {
	return a.addressFormat
}

func (a *Address) SetDerivationType(dt string) error {
	dt = strings.TrimSpace(dt)
	dt = strings.ToLower(dt)

	if a.path == nil {
		a.path = &DerivationPath{}
	}

	if dt != `` {
		if _, ok := derivationIndex[DerivationType(dt)]; !ok {
			return errors.New(fmt.Sprintf(`"dt" param has wrong value "%s"`, dt))
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
		return errors.New(`incorrect "dp" param`)
	}

	dp = strings.TrimSpace(dp)
	dp = strings.ToLower(dp)

	if !rx.MatchString(dp) {
		return errors.New(fmt.Sprintf(`"dp" param has wrong value "%s"`, dp))
	}

	return a.path.ParsePath(dp)
}

func (a *Address) SetCoinType(ct string) error {
	ct = strings.TrimSpace(ct)

	// TODO: Check coin type extraction from derivation path??? subnets???
	if ct == `` {
		return errors.New(fmt.Sprintf(`"ct" required for "ct=%s"`, a.chain.networkType))
	}

	coinType, err := strconv.ParseUint(ct, 0, 32)
	if err != nil {
		return errors.New(`cannot parse "ct"`)
	}

	a.chain.coinType = CoinType(coinType)

	return nil
}

func (a *Address) SetAddressAlgorithm(aa string) error {
	aa = strings.TrimSpace(aa)
	aa = strings.ToLower(aa)
	if aa == `` {
		// set default
		switch a.chain.networkType {
		case Bitcoin, EthereumVM, AvalancheVM, TronVM, Cosmos:
			a.addressAlgorithm = Secp256k1
		case Solana:
			a.addressAlgorithm = Ed25519
		}
	} else {
		if _, ok := indexAlgorithms[Algorithm(aa)]; !ok {
			return errors.New(`incorrect "aa" param`)
		}
		a.addressAlgorithm = Algorithm(aa)
	}

	return nil
}

func (a *Address) SetAddressFormat(af string) error {
	af = strings.TrimSpace(af)
	if af != `` {
		a.addressFormat = Format(af)
	}
	return nil
}

func (a *Address) SetAddressPrefix(ap string) error {
	ap = strings.TrimSpace(ap)
	if ap != `` {
		a.addressPrefix = ap
	}

	return nil
}

func (a *Address) SetAddressSuffix(as string) error {
	as = strings.TrimSpace(as)
	if as != `` {
		a.addressSuffix = as
	}
	return nil
}

func (a *Address) String() string {
	return fmt.Sprintf(`urn:mhda:%s`, a.NSS())
}

func (a *Address) NSS() string {
	result := fmt.Sprintf(`nt:%s`, a.chain.networkType)

	if a.path.derivationType != ROOT {
		result += fmt.Sprintf(`:dt:%s:dp:%s`, a.path.derivationType, a.path.String())
	}

	result += fmt.Sprintf(`:ct:%d:ci:%s`, a.chain.coinType, a.chain.chainId)

	// TODO: add full mode
	// TODO: use additional params, when address has non-default values

	return result
}

func (a *Address) Hash() string {
	h := sha1.New()
	h.Write([]byte(a.String()))
	return hex.EncodeToString(h.Sum(nil))
}

func (a *Address) NSSHash() string {
	h := sha1.New()
	h.Write([]byte(a.NSS()))
	return hex.EncodeToString(h.Sum(nil))
}
