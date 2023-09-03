package go_mhda

import (
	"errors"
	"regexp"
	"strings"
)

const (
	prefixMHDA          = `urn:mhda:`
	prefixOffset        = 9
	indexComponentIndex = 1
	indexComponentValue = 2

	compNetworkType      = `nt`
	compDerivationType   = `dt`
	compDerivationPath   = `dp`
	compCoinType         = `ct`
	compChainId          = `ci`
	compAddressAlgorithm = `aa`
	compAddressFormat    = `ad`
	compAddressPrefix    = `ap`
	compAddressSuffix    = `as`
)

var (
	componentsNames = []string{`nt`, `dt`, `dp`, `ct`, `ci`, `aa`, `af`, `ap`, `as`}

	rxComponent = regexp.MustCompile(`:(nt|dt|dp|ct|ci|aa|af|ap|as):([0-9a-z-._~*+=%$&@?'()!,;/#]+)`)
)

type urn struct {
	nid        string
	components map[string]string
}

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

func ParseNSS(nss string) (MHDA, error) {
	var componentsNamesTmp = make([]string, len(componentsNames))

	copy(componentsNamesTmp, componentsNames)

	components := map[string]string{}

	iter := 0

	for iter < len(nss) {
		var isFound bool
		for i := range componentsNamesTmp {
			if nss[iter] == componentsNamesTmp[i][0] && nss[iter+1] == componentsNamesTmp[i][1] {
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
						nss[iterVal] == 33 || // 33 [!]
						nss[iterVal] == 59 || // 59 [;]
						nss[iterVal] == 61 || // 61 [=]
						nss[iterVal] == 63 || // 63 [?]
						nss[iterVal] == 64 { // 64 [@]

						isFound = true
						componentValue += nss[iterVal : iterVal+1]
						iterVal++
					}
				}
				components[componentIndex] = componentValue
				if isFound {
					componentsNamesTmp = append(componentsNamesTmp[:i], componentsNamesTmp[i+1:]...)
					iter = iterVal + 1
					break
				}
			}
		}
		if !isFound {
			iter += 3
		}
	}

	if _, ok := components[compNetworkType]; !ok {
		return nil, errors.New(`"nt" not defined`)
	}

	return parseAddress(components)
}
