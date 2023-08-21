package go_mhda

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

const (
	ROOT  = DerivationType(`root`)
	BIP32 = DerivationType(`bip32`)
	BIP44 = DerivationType(`bip44`)
	BIP84 = DerivationType(`bip84`)
	CIP11 = DerivationType(`cip11`)
	ZIP32 = DerivationType(`zip32`)
)

type DerivationType string

type AccountIndex uint32

type ChargeType uint8

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
}

func (p *DerivationPath) DerivationType() DerivationType {
	return p.derivationType
}

func (p *DerivationPath) Coin() CoinType {
	return p.coin
}

func (p *DerivationPath) Account() AccountIndex {
	return p.account
}

func (p *DerivationPath) Charge() ChargeType {
	return p.charge
}

func (p *DerivationPath) Index() AddressIndex {
	return p.index
}

func (p *DerivationPath) IsHardenedAddress() bool {
	return p.index.IsHardened
}

var (
	rxRoot = regexp.MustCompile("")

	// https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
	// m / account ' / charge / address
	rxBip32 = regexp.MustCompile(`^m/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki
	// m / 44 ' / coin ' / account ' / charge / address
	rxBip44 = regexp.MustCompile(`^m/44[Hh']/([0-9]+)[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/bitcoin/bips/blob/master/bip-0084.mediawiki
	// m / 84 ' / 0 ' / account ' / charge / address
	rxBip84 = regexp.MustCompile(`^m/84[Hh']/0[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/confio/cosmos-hd-key-derivation-spec
	// m / 44 ' / 118 ' / account ' / charge_extra / address
	rxCip11 = regexp.MustCompile(`^m/44[Hh']/118[Hh']/([0-9]+)[Hh']/([0-9]+)/([0-9]+)([Hh'])?$`)

	// https://zips.z.cash/zip-0032
	// m / 32 ' / 133 ' / account '
	// m / 32 ' / 133 ' / account ' / address
	// m / 32 ' / 133 ' / account ' / address '
	rxZip32 = regexp.MustCompile("")

	derivationIndex = map[DerivationType]*regexp.Regexp{
		ROOT:  rxRoot,
		BIP32: rxBip32,
		BIP44: rxBip44,
		BIP84: rxBip84,
		CIP11: rxCip11,
		ZIP32: rxZip32,
	}
)

func ParsePath(dt DerivationType, path string) (*DerivationPath, error) {
	var isAddressHardened = false
	matches := derivationIndex[dt].FindStringSubmatch(path)
	if len(matches) < 5 {
		return nil, errors.New(fmt.Sprintf("cannot parse path: %s", path))
	}
	networkType, err := strconv.ParseUint(matches[1], 10, 32)
	if err != nil {
		return nil, err
	}
	accountIndex, err := strconv.ParseUint(matches[2], 10, 32)
	if err != nil {
		return nil, err
	}
	chargeType, err := strconv.ParseUint(matches[3], 10, 32)
	if err != nil {
		return nil, err
	}
	addressIndex, err := strconv.ParseUint(matches[4], 10, 32)
	if err != nil {
		return nil, err
	}
	if len(matches) == 6 && matches[5] != "" {
		isAddressHardened = true
	}

	return &DerivationPath{
		derivationType: dt,
		coin:           CoinType(networkType),
		account:        AccountIndex(accountIndex),
		charge:         ChargeType(chargeType),
		index: AddressIndex{
			Index:      uint32(addressIndex),
			IsHardened: isAddressHardened,
		},
	}, nil
}

func (p *DerivationPath) String() string {
	var result string

	switch p.derivationType {
	case ROOT:
		return ``
	case BIP32:
		var format = "m/%d'/%d/%d"
		if p.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, p.account, p.charge, p.index.Index)
	case BIP44:
		var format = "m/44'/%d'/%d'/%d/%d"
		if p.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, p.coin, p.account, p.charge, p.index.Index)
	case BIP84:
		var format = "m/84'/%d'/%d'/%d/%d"
		if p.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, p.coin, p.account, p.charge, p.index.Index)
	case CIP11:
		var format = "m/44'/133'/%d'/%d/%d"
		if p.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, p.account, p.charge, p.index.Index)
	case ZIP32:
		var format = "m/32'/133'/%d'/%d"
		if p.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, p.account, p.index.Index)
	}

	return result
}
