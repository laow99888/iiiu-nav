package linkmeta

import (
	"context"
	"net"
	"net/netip"
	"net/url"
	"time"
)

type ipResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

func validatePublicURL(ctx context.Context, target *url.URL) error {
	return validateURLWithResolver(ctx, target, net.DefaultResolver)
}

func validateURLWithResolver(ctx context.Context, target *url.URL, resolver ipResolver) error {
	if target == nil || target.User != nil || target.Hostname() == "" || (target.Scheme != "http" && target.Scheme != "https") {
		return ErrUnsafeURL
	}
	_, err := resolveAllowedAddresses(ctx, resolver, target.Hostname())
	return err
}

func resolveAllowedAddresses(ctx context.Context, resolver ipResolver, host string) ([]netip.Addr, error) {
	if address, err := netip.ParseAddr(host); err == nil {
		if !isPublicIP(address) {
			return nil, ErrUnsafeURL
		}
		return []netip.Addr{address}, nil
	}
	addresses, err := resolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return nil, ErrUnsafeURL
	}
	for _, address := range addresses {
		if !isPublicIP(address) && !proxyFakeIPPrefix.Contains(address.Unmap()) {
			return nil, ErrUnsafeURL
		}
	}
	return addresses, nil
}

func publicDialer(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, ErrUnsafeURL
	}
	addresses, err := resolveAllowedAddresses(ctx, net.DefaultResolver, host)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: 4 * time.Second, KeepAlive: 30 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
}

func isPublicIP(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

var proxyFakeIPPrefix = netip.MustParsePrefix("198.18.0.0/15")
