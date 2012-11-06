package conntrack

import (
	"net"
)

type FilterFlag uint8

const (
	SNATFilter FilterFlag = 1 << iota
	DNATFilter
	RoutedFilter
	LocalFilter
)

var localIPs = make([]*net.IPNet, 0)

func isLocalIP(ip net.IP) bool {
	for _, localIP := range localIPs {
		if localIP.IP.Equal(ip) {
			return true
		}
	}

	return false
}

func init() {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}

	for _, address := range addresses {
		localIPs = append(localIPs, address.(*net.IPNet))
	}
}

func FilterSNAT(flows []Flow) []Flow {
	return Filter(flows, SNATFilter)
}

func FilterDNAT(flows []Flow) []Flow {
	return Filter(flows, DNATFilter)
}

func FilterRouted(flows []Flow) []Flow {
	return Filter(flows, RoutedFilter)
}

func FilterLocal(flows []Flow) []Flow {
	return Filter(flows, LocalFilter)
}

func Filter(flows []Flow, which FilterFlag) []Flow {
	natFlows := make([]Flow, 0, len(flows))

	snat := (which & SNATFilter) > 0
	dnat := (which & DNATFilter) > 0
	local := (which & LocalFilter) > 0
	routed := (which & RoutedFilter) > 0

	for _, flow := range flows {
		if (snat && isSNAT(flow)) ||
			(dnat && isDNAT(flow)) ||
			(local && isLocal(flow)) ||
			(routed && isRouted(flow)) {

			natFlows = append(natFlows, flow)
		}
	}

	return natFlows
}

func isSNAT(flow Flow) bool {
	// SNATed flows should reply to our WAN IP, not a LAN IP.
	if flow.Original.Source.Equal(flow.Reply.Destination) {
		return false
	}

	if !flow.Original.Destination.Equal(flow.Reply.Source) {
		return false
	}

	return true
}

func isDNAT(flow Flow) bool {
	// Reply must go back to the source; Reply mustn't come from the WAN IP
	if flow.Original.Source.Equal(flow.Reply.Destination) && !flow.Original.Destination.Equal(flow.Reply.Source) {
		return true
	}

	// Taken straight from original netstat-nat, labelled "DNAT (1 interface)"
	if !flow.Original.Source.Equal(flow.Reply.Source) && !flow.Original.Source.Equal(flow.Reply.Destination) && !flow.Original.Destination.Equal(flow.Reply.Source) && flow.Original.Destination.Equal(flow.Reply.Destination) {
		return true
	}

	return false
}

func isLocal(flow Flow) bool {
	// no NAT
	if flow.Original.Source.Equal(flow.Reply.Destination) && flow.Original.Destination.Equal(flow.Reply.Source) {
		// At least one local address
		if isLocalIP(flow.Original.Source) || isLocalIP(flow.Original.Destination) || isLocalIP(flow.Reply.Source) || isLocalIP(flow.Reply.Destination) {
			return true
		}
	}

	return false
}

func isRouted(flow Flow) bool {
	// no NAT
	if flow.Original.Source.Equal(flow.Reply.Destination) && flow.Original.Destination.Equal(flow.Reply.Source) {
		// No local addresses
		if !isLocalIP(flow.Original.Source) && !isLocalIP(flow.Original.Destination) && !isLocalIP(flow.Reply.Source) && !isLocalIP(flow.Reply.Destination) {
			return true
		}
	}

	return false
}
