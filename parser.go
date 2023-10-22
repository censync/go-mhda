package go_mhda

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	prefixMHDA          = `urn:mhda:`
	prefixOffset        = 9
	indexComponentIndex = 1
	indexComponentValue = 2

	// NSS components

	// Chain domain

	// compNetworkType is Network Type description, e.g. "evm", "tvm", "avm", "btc", "cosmos"
	compNetworkType = `nt`
	// compCoinType is Coin Type description, according SLIP-44 list (https://github.com/satoshilabs/slips/blob/master/slip-0044.md)
	// e.g. "0", "60", "195", "118"
	compCoinType = `ct`
	// compChainId is Network Id (Chain Id) description
	// e.g  for evm hex: "0x1", "0x10", for Cosmos - string "axelar", etc
	compChainId = `ci`

	// Derivation path domain
	compDerivationType = `dt`
	compDerivationPath = `dp`

	// Address format domain
	compAddressAlgorithm = `aa`
	compAddressFormat    = `ad`
	compAddressPrefix    = `ap`
	compAddressSuffix    = `as`
)

var (
	componentsNames = []string{
		compNetworkType,
		compDerivationType,
		compDerivationPath,
		compCoinType,
		compChainId,
		compAddressAlgorithm,
		compAddressFormat,
		compAddressPrefix,
		compAddressSuffix,
	}

	rxComponent = regexp.MustCompile(`:(nt|ct|ci|dt|dp|aa|af|ap|as):([0-9a-z-._~*+=%$&@?'()!,;/#]+)`)
)

// TODO: Match to RFC 8141

func ParseURNRx(src string) (MHDA, error) {
	if !strings.HasPrefix(src, prefixMHDA) {
		return nil, errors.New("source string is not valid URN MHDA")
	}

	submatches := rxComponent.FindAllStringSubmatch(src[prefixOffset-1:], len(componentsNames))

	if len(submatches) == 0 {
		return nil, errors.New("no components")
	}

	components := map[string]string{}

	for i := range submatches {
		if len(submatches[i]) != 3 {
			continue
		}
		components[submatches[i][indexComponentIndex]] = submatches[i][indexComponentValue]
	}

	//log.Println(components)

	return nil, nil
}
func ParseURN(src string) (MHDA, error) {
	if !strings.HasPrefix(src, prefixMHDA) {
		return nil, errors.New("source string is not valid URN MHDA")
	}

	return ParseNSS(src[prefixOffset:])
}

func ParseNSS(src string) (MHDA, error) {
	var componentsNamesTmp = make([]string, len(componentsNames))

	copy(componentsNamesTmp, componentsNames)

	components, err := parseNSS(src, componentsNamesTmp)

	if err != nil {
		return nil, err
	}

	if _, ok := components[compNetworkType]; !ok {
		return nil, errors.New(`"nt" not defined`)
	}

	return parseAddress(components)
}

func parseNSS(nss string, components []string) (map[string]string, error) {
	result := map[string]string{}

	iter := 0

	for iter < len(nss) {
		var isFound bool
		for i := range components {
			if nss[iter] == components[i][0] && nss[iter+1] == components[i][1] {
				componentIndex := nss[iter : iter+2]
				componentValue := ``
				iterVal := iter + 3
				for iterVal < len(nss) {
					if nss[iterVal] == 58 { // 58 [:]  separator
						break
					}

					// ASCII checks instead regexp for performance

					if (nss[iterVal] >= 48 && nss[iterVal] <= 57) || // 48-57  [0-9]
						(nss[iterVal] >= 97 && nss[iterVal] <= 122) || // 97-122 [a-z]
						(nss[iterVal] >= 35 && nss[iterVal] <= 47) || // 35-47 [!#$%&'()*+,-./]
						(nss[iterVal] >= 65 && nss[iterVal] <= 90) || // 65-90 [A-Z]
						nss[iterVal] == 95 || // 95 [_]
						nss[iterVal] == 33 || // 33 [!]
						nss[iterVal] == 59 || // 59 [;]
						nss[iterVal] == 61 || // 61 [=]
						nss[iterVal] == 63 || // 63 [?]
						nss[iterVal] == 64 { // 64 [@]

						isFound = true
						componentValue += nss[iterVal : iterVal+1]
						iterVal++
					} else {
						return nil, fmt.Errorf("cannot parse nss: wrong symbol %q, pos %d", nss[iterVal], iterVal)
					}

				}
				result[componentIndex] = componentValue
				if isFound {
					components = append(components[:i], components[i+1:]...)
					iter = iterVal + 1
					break
				}
			}
		}
		if !isFound {
			iter += 3
		}
	}
	return result, nil
}
