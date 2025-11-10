package api

import "net"

// isIPv6 checks if the given string is an IPv6 address
func isIPv6(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil && ip.To4() == nil
}
