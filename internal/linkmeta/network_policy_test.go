package linkmeta

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"testing"
)

type resolverFunc func(context.Context, string, string) ([]netip.Addr, error)

func (function resolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return function(ctx, network, host)
}

func TestURLPolicyAllowsProxyFakeIPOnlyForDomainNames(t *testing.T) {
	t.Parallel()
	resolver := resolverFunc(func(_ context.Context, _, host string) ([]netip.Addr, error) {
		switch host {
		case "public-via-proxy.test":
			return []netip.Addr{netip.MustParseAddr("198.18.0.12")}, nil
		case "private.test":
			return []netip.Addr{netip.MustParseAddr("192.168.1.10")}, nil
		default:
			return nil, errors.New("unexpected host")
		}
	})

	for _, test := range []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{name: "domain resolved through fake IP proxy", rawURL: "https://public-via-proxy.test"},
		{name: "explicit fake IP", rawURL: "https://198.18.0.12", wantErr: true},
		{name: "domain resolved to private IP", rawURL: "https://private.test", wantErr: true},
		{name: "explicit loopback IP", rawURL: "http://127.0.0.1", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			target, err := url.Parse(test.rawURL)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			err = validateURLWithResolver(context.Background(), target, resolver)
			if test.wantErr && !errors.Is(err, ErrUnsafeURL) {
				t.Fatalf("validateURLWithResolver() error = %v, want ErrUnsafeURL", err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("validateURLWithResolver() error = %v", err)
			}
		})
	}
}
