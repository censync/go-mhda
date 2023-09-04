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

	ChargeExternal = ChargeType(0)
	ChargeInternal = ChargeType(1)
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

func NewDerivationPath(derivationType DerivationType, coin CoinType, account AccountIndex, charge ChargeType, index AddressIndex) *DerivationPath {
	return &DerivationPath{
		derivationType: derivationType,
		coin:           coin,
		account:        account,
		charge:         charge,
		index:          index,
	}
}

func ParseDerivationPath(dt DerivationType, path string) (*DerivationPath, error) {
	rx, ok := derivationIndex[dt]

	if !ok {
		return nil, errors.New("wrong derivation type")
	}

	if !rx.MatchString(path) {
		return nil, errors.New("incorrect derivation path")
	}

	dPath := &DerivationPath{
		derivationType: dt,
	}

	err := dPath.ParsePath(path)

	if err != nil {
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
	rxRoot = regexp.MustCompile("")

	// https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
	// m / account ' / charge / address
	rxBip32 = regexp.MustCompile(`^m/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki
	// m / 44 ' / coin ' / account ' / charge / address
	rxBip44 = regexp.MustCompile(`^m/44[Hh']/([0-9]+)[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)([Hh'])?$`)

	// https://github.com/bitcoin/bips/blob/master/bip-0084.mediawiki
	// m / 84 ' / 0 ' / account ' / charge / address
	// TODO: Check for hardened index
	rxBip84 = regexp.MustCompile(`^m/84[Hh']/0[Hh']/([0-9]+)[Hh']/(0|1)/([0-9]+)?$`)

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

func (dp *DerivationPath) ParsePath(path string) error {
	var isAddressHardened = false
	matches := derivationIndex[dp.derivationType].FindStringSubmatch(path)
	// TODO: Fix serialization for different length
	if len(matches) < 5 {
		return errors.New(fmt.Sprintf("cannot parse path: %s", path))
	}
	coinType, err := strconv.ParseUint(matches[1], 10, 32)
	if err != nil {
		return errors.New("cannot parse coin")
	}
	accountIndex, err := strconv.ParseUint(matches[2], 10, 32)
	if err != nil {
		return errors.New("cannot parse account")
	}
	chargeType, err := strconv.ParseUint(matches[3], 10, 32)
	if err != nil {
		return errors.New("cannot parse charge")
	}
	addressIndex, err := strconv.ParseUint(matches[4], 10, 32)
	if err != nil {
		return errors.New("cannot parse index")
	}
	if len(matches) == 6 && matches[5] != "" {
		isAddressHardened = true
	}

	dp.coin = CoinType(coinType)
	dp.account = AccountIndex(accountIndex)
	dp.charge = ChargeType(chargeType)
	dp.index = AddressIndex{
		Index:      uint32(addressIndex),
		IsHardened: isAddressHardened,
	}

	return nil
}

func (dp *DerivationPath) String() string {
	var result string

	switch dp.derivationType {
	case ROOT:
		return ``
	case BIP32:
		var format = "m/%d'/%d/%d"
		if dp.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, dp.account, dp.charge, dp.index.Index)
	case BIP44:
		var format = "m/44'/%d'/%d'/%d/%d"
		if dp.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, dp.coin, dp.account, dp.charge, dp.index.Index)
	case BIP84:
		var format = "m/84'/%d'/%d'/%d/%d"
		if dp.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, dp.coin, dp.account, dp.charge, dp.index.Index)
	case CIP11:
		var format = "m/44'/133'/%d'/%d/%d"
		if dp.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, dp.account, dp.charge, dp.index.Index)
	case ZIP32:
		var format = "m/32'/133'/%d'/%d"
		if dp.index.IsHardened {
			format += `'`
		}
		return fmt.Sprintf(format, dp.account, dp.index.Index)
	}

	return result
}
