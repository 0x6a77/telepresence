package tun

import (
	"errors"
	"fmt"
	"net"
)

type ipAndNetwork struct {
	ip      net.IP
	network *net.IPNet
}

// stubbable version
var interfaceAddrs = net.InterfaceAddrs

// findAvailableSubnetClassC returns the first class C subnet CIDR in the address ranges reserved
// for private (non-routed) use that isn't in use by an existing network interface.
func findAvailableSubnetClassC() (string, error) {
	addrs, err := interfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("failed to obtain interface addresses: %v", err)
	}

	cidrs := make([]*ipAndNetwork, 0, len(addrs))
	for _, a := range addrs {
		if ip, network, err := net.ParseCIDR(a.String()); err == nil {
			cidrs = append(cidrs, &ipAndNetwork{ip: ip, network: network})
		}
	}

	findChunk := func(ar1, ar2 int) int {
		_, wantedRange, err := net.ParseCIDR(fmt.Sprintf("%d.%d.0.0/16", ar1, ar2))
		if err != nil {
			panic(err)
		}
		return findAvailableChunk(wantedRange, cidrs)
	}

	cidr24 := func(ar1, ar2, ar3 int) string {
		return fmt.Sprintf("%d.%d.%d.0/24", ar1, ar2, ar3)
	}

	for i := 0; i < 256; i++ {
		if found := findChunk(10, i); found >= 0 {
			return cidr24(10, i, found), nil
		}
	}
	for i := 16; i < 32; i++ {
		if found := findChunk(17, i); found >= 0 {
			return cidr24(17, i, found), nil
		}
	}
	if found := findChunk(192, 168); found >= 0 {
		return cidr24(192, 168, found), nil
	}
	return "", errors.New("no available CIDR")
}

// covers answers the question if network range a contains all of network range b
func covers(a, b *net.IPNet) bool {
	if !a.Contains(b.IP) {
		return false
	}

	// create max IP in range b using its mask
	ones, _ := b.Mask.Size()
	l := len(b.IP)
	m := make(net.IP, l)
	n := uint(ones)
	for i := 0; i < l; i++ {
		switch {
		case n >= 8:
			m[i] = b.IP[i]
			n -= 8
		case n > 0:
			m[i] = b.IP[i] | byte(0xff>>n)
			n = 0
		default:
			m[i] = 0xff
		}
	}
	return a.Contains(m)
}

func findAvailableChunk(wantedRange *net.IPNet, cidrs []*ipAndNetwork) int {
	inUse := [256]bool{}
	for _, cid := range cidrs {
		if covers(cid.network, wantedRange) {
			return -1
		}
		if !wantedRange.Contains(cid.ip) {
			continue
		}
		ones, bits := cid.network.Mask.Size()
		if bits != 32 {
			return -1
		}
		if ones >= 24 {
			inUse[cid.network.IP[2]] = true
		} else {
			ones -= 16
			mask := 0xff >> ones
			for i := 0; i <= mask; i++ {
				inUse[i] = true
			}
		}
	}

	for i := 0; i < 256; i++ {
		if !inUse[i] {
			return i
		}
	}
	return -1
}
