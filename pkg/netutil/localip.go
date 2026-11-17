package netutil

import "net"

// LocalIP returns the first non-loopback IPv4 address used for outbound
// traffic. No packets are actually sent: UDP dial only selects a route.
func LocalIP() (string, error) {
	conn, err := net.Dial("udp", "203.0.113.1:80") // documentation range
	if err != nil {
		return "", err
	}
	defer conn.Close()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || addr.IP == nil {
		return "", net.ErrClosed
	}
	return addr.IP.String(), nil
}
