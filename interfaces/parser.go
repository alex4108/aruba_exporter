package interfaces

import (
	"errors"
	"regexp"
	"strings"

	"github.com/yankiwi/aruba_exporter/rpc"
	"github.com/yankiwi/aruba_exporter/util"
	
	"github.com/prometheus/common/log"
)

// Parse parses cli output and tries to find interfaces with related stats
func (c *interfaceCollector) Parse(ostype string, output string) ([]Interface, error) {
	log.Debugf("OS: %s\n", ostype)
	log.Debugf("output: %s\n", output)
	if ostype != rpc.ArubaSwitch {
		return nil, errors.New("'show interface' is not implemented for " + ostype)
	}
	items := []Interface{}
	newIfRegexp := regexp.MustCompile(`^\s+Status and Counters - Port Counters for port (\d+\/?\d*)\s*$`)
	descRegexp := regexp.MustCompile(`^\s+Name\s+:\s+(.*)$`)
	macRegexp := regexp.MustCompile(`^\s+MAC Address\s+:\s+(.*)$`)
	linkStatusRegexp := regexp.MustCompile(`^\s+Link Status\s+:\s+(Up|Down)\s*$`)
	portEnabledRegexp := regexp.MustCompile(`^\s+Port Enabled\s+:\s+(Yes|No)\s*$`)
	bytesRegexp := regexp.MustCompile(`\s+Bytes Rx\s+:\s+(\d+)\s+Bytes Tx\s+:\s+(\d+)\s*$`)
	unicastRegexp := regexp.MustCompile(`\s+Unicast Rx\s+:\s+(\d+)\s+Unicast Tx\s+:\s+(\d+)\s*$`)
	BandMcastRegexp := regexp.MustCompile(`\s+Bcast\/Mcast Rx\s+:\s+(\d+)\s+Bcast\/Mcast R\Tx\s+:\s+(\d+)\s*$`)
	RxDrops := regexp.MustCompile(`\s+Bcast\/Mcast Rx\s+:\s+(\d+)\s+Bcast\/Mcast R\Tx\s+:\s+(\d+)\s*$`)
	TxDrops
	RxErrors
	
	current := Interface{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if newIfRegexp.MatchString(line) {
			if current != (Interface{}) {
				items = append(items, current)
			}
			current = Interface{
				Name: matches[1],
			}
			continue
		}

		if matches := descRegexp.FindStringSubmatch(line); matches != nil {
			current.Description = matches[1]
			continue
		}

		if matches := macRegexp.FindStringSubmatch(line); matches != nil {
			current.MacAddress = matches[1]
			continue
		}

		if matches := linkStatusRegexp.FindStringSubmatch(line); matches != nil {
			current.linkStatus = matches[1]
			continue
		}

		if matches := portEnabledRegexp.FindStringSubmatch(line); matches != nil {
			current.portEnabled = matches[1]
			continue
		}

		if matches := bytesRegexp.FindStringSubmatch(line); matches != nil {
			current.RxBytes = util.Str2float64(matches[1])
			current.TxBytes = util.Str2float64(matches[2])
			continue
		}

		if matches := unicastRegexp.FindStringSubmatch(line); matches != nil {
			current.RxUnicast = util.Str2float64(matches[1])
			current.TxUnicast = util.Str2float64(matches[2])
			continue
		}

		if matches := BandMcastRegexp.FindStringSubmatch(line); matches != nil {
			current.RxBandMcast = util.Str2float64(matches[1])
			current.TxBandMcast = util.Str2float64(matches[2])
			continue
		}


	}
	return append(items, current), nil
}

// ParseVlans parses cli output and tries to find vlans with related traffic stats
func (c *interfaceCollector) ParseVlans(ostype string, output string) ([]Interface, error) {
	log.Debugf("OS: %s\n", ostype)
	log.Debugf("output: %s\n", output)
	if ostype != rpc.ArubaSwitch {
		return nil, errors.New("'show vlans' is not implemented for " + ostype)
	}
	items := []Interface{}
	deviceNameRegexp, _ := regexp.Compile(`^([a-zA-Z0-9\/-]+\.[a-zA-Z0-9\/-]+) \(:?\d+\).*$`)
	inputBytesRegexp, _ := regexp.Compile(`^\s+Total \d+ packets, (\d+) bytes input.*$`)
	outputBytesRegexp, _ := regexp.Compile(`^\s+Total \d+ packets, (\d+) bytes output.*$`)

	current := Interface{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if matches := deviceNameRegexp.FindStringSubmatch(line); matches != nil {
			if current != (Interface{}) {
				items = append(items, current)
			}
			current = Interface{
				Name: matches[1],
			}
		}
		if current == (Interface{}) {
			continue
		}
		if matches := inputBytesRegexp.FindStringSubmatch(line); matches != nil {
			current.InputBytes = util.Str2float64(matches[1])
		} else if matches := outputBytesRegexp.FindStringSubmatch(line); matches != nil {
			current.OutputBytes = util.Str2float64(matches[1])
		}
	}
	return append(items, current), nil
}