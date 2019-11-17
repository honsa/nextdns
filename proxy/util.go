package proxy

import (
	"net"
	"strconv"
	"strings"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/nextdns/nextdns/resolver"
)

func replyNXDomain(q resolver.Query, buf []byte) (n int, err error) {
	var p dnsmessage.Parser
	h, err := p.Start(q.Payload)
	if err != nil {
		return 0, err
	}
	q1, err := p.Question()
	if err != nil {
		return 0, err
	}
	h.Response = true
	h.RCode = dnsmessage.RCodeNameError
	b := dnsmessage.NewBuilder(buf[:0], h)
	_ = b.Question(q1)
	buf, err = b.Finish()
	return len(buf), err
}

func isPrivateReverse(qname string) bool {
	if ip := ptrIP(qname); ip != nil {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			return true
		}
		if ip := ip.To4(); ip != nil {
			return (ip[0] == 10) ||
				(ip[0] == 172 && ip[1]&0xf0 == 16) ||
				(ip[0] == 192 && ip[1] == 168)
		}
		return ip[0] == 0xfd
	}
	return false
}

func ptrIP(ptr string) net.IP {
	if !strings.HasSuffix(ptr, ".arpa.") {
		return nil
	}
	ptr = ptr[:len(ptr)-6]
	var l int
	var base int
	if strings.HasSuffix(ptr, ".in-addr") {
		ptr = ptr[:len(ptr)-8]
		l = net.IPv4len
		base = 10
	} else if strings.HasSuffix(ptr, ".ip6") {
		ptr = ptr[:len(ptr)-4]
		l = net.IPv6len
		base = 16
	}
	if l == 0 {
		return nil
	}
	ip := make(net.IP, l)
	if base == 16 {
		l *= 2
	}
	for i := 0; i < l && ptr != ""; i++ {
		idx := strings.LastIndexByte(ptr, '.')
		off := idx + 1
		if idx == -1 {
			idx = 0
			off = 0
		} else if idx == len(ptr)-1 {
			return nil
		}
		n, err := strconv.ParseUint(ptr[off:], base, 8)
		if err != nil {
			return nil
		}
		b := byte(n)
		ii := i
		if base == 16 {
			// ip6 use hex nibbles instead of base 10 bytes, so we need to join
			// nibbles by two.
			ii /= 2
			if i&1 == 1 {
				b |= ip[ii] << 4
			}
		}
		ip[ii] = b
		ptr = ptr[:idx]
	}
	return ip
}

func addrIP(addr net.Addr) (ip net.IP) {
	// Avoid parsing/alloc when it's an IP already.
	switch addr := addr.(type) {
	case *net.IPAddr:
		ip = addr.IP
	case *net.UDPAddr:
		ip = addr.IP
	case *net.TCPAddr:
		ip = addr.IP
	default:
		host, _, _ := net.SplitHostPort(addr.String())
		ip = net.ParseIP(host)
	}
	return
}
