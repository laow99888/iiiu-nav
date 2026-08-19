package linkmeta

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (function doerFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

func allowHTTP(_ context.Context, target *url.URL) error {
	if target.Scheme != "http" && target.Scheme != "https" {
		return ErrUnsafeURL
	}
	return nil
}

func TestRecognizeExtractsMetadataAndDeclaredIcon(t *testing.T) {
	t.Parallel()
	client := doerFunc(func(request *http.Request) (*http.Response, error) {
		var body, contentType string
		switch request.URL.Path {
		case "/page":
			body = `<html><head><title>  Example   Tools </title><meta name="description" content="Useful tools"><link rel="icon" href="/assets/icon.png"></head></html>`
			contentType = "text/html; charset=utf-8"
		case "/assets/icon.png":
			body = "image bytes"
			contentType = "image/png"
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: request}, nil
		}
		header := make(http.Header)
		header.Set("Content-Type", contentType)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: header, Request: request}, nil
	})
	result, err := NewWithClient(client, allowHTTP).Recognize(context.Background(), "https://example.test/page")
	if err != nil {
		t.Fatalf("recognize: %v", err)
	}
	if result.Title != "Example Tools" || result.Description != "Useful tools" {
		t.Fatalf("unexpected metadata: %#v", result)
	}
	if string(result.Icon) != "image bytes" || result.IconType != "image/png" {
		t.Fatalf("unexpected icon: %#v", result)
	}
}

func TestRecognizeFallsBackToRootFavicon(t *testing.T) {
	t.Parallel()
	requested := make([]string, 0, 2)
	client := doerFunc(func(request *http.Request) (*http.Response, error) {
		requested = append(requested, request.URL.Path)
		header := make(http.Header)
		if request.URL.Path == "/nested/page" {
			header.Set("Content-Type", "text/html")
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`<meta property="og:title" content="Fallback">`)), Header: header, Request: request}, nil
		}
		header.Set("Content-Type", "image/x-icon")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ico")), Header: header, Request: request}, nil
	})
	result, err := NewWithClient(client, allowHTTP).Recognize(context.Background(), "https://example.test/nested/page")
	if err != nil {
		t.Fatalf("recognize: %v", err)
	}
	if result.Title != "Fallback" || string(result.Icon) != "ico" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(requested) != 2 || requested[1] != "/favicon.ico" {
		t.Fatalf("unexpected requests: %#v", requested)
	}
}

func TestParseHTMLUsesNestedAndSocialMetadataFallbacks(t *testing.T) {
	t.Parallel()
	result, _, err := parseHTML([]byte(`<html><head><meta name="twitter:title" content="Social title"><meta name="twitter:description" content="Social description"></head><body><h1>Ignored fallback</h1></body></html>`), mustURL(t, "https://example.test"))
	if err != nil {
		t.Fatalf("parse HTML: %v", err)
	}
	if result.Title != "Social title" || result.Description != "Social description" {
		t.Fatalf("unexpected metadata: %#v", result)
	}
}

func TestDecodeHTMLHonorsDeclaredLegacyCharset(t *testing.T) {
	t.Parallel()
	content := []byte("<title>\xB2\xE2\xCA\xD4</title>")
	decoded, err := decodeHTML(content, "text/html; charset=gbk")
	if err != nil {
		t.Fatalf("decode HTML: %v", err)
	}
	if !strings.Contains(string(decoded), "测试") {
		t.Fatalf("decoded content does not contain expected text: %q", decoded)
	}
}

func mustURL(t *testing.T, value string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return parsed
}

func TestRecognizeRejectsInvalidContentAndOversizedHTML(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		contentType string
		body        []byte
	}{
		{name: "non html", contentType: "application/json", body: []byte(`{}`)},
		{name: "too large", contentType: "text/html", body: bytes.Repeat([]byte("x"), maxHTMLBytes+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := doerFunc(func(request *http.Request) (*http.Response, error) {
				header := make(http.Header)
				header.Set("Content-Type", test.contentType)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(test.body)), Header: header, Request: request}, nil
			})
			if _, err := NewWithClient(client, allowHTTP).Recognize(context.Background(), "https://example.test"); err == nil {
				t.Fatal("expected recognition to fail")
			}
		})
	}
}

func TestPublicIPAddressPolicy(t *testing.T) {
	t.Parallel()
	for value, expected := range map[string]bool{
		"8.8.8.8": true, "1.1.1.1": true, "10.0.0.1": false,
		"127.0.0.1": false, "169.254.169.254": false, "100.64.0.1": false,
		"192.0.2.1": false, "198.18.0.1": false, "::1": false,
		"fc00::1": false, "2001:db8::1": false, "2606:4700:4700::1111": true,
	} {
		if actual := isPublicIP(netip.MustParseAddr(value)); actual != expected {
			t.Errorf("isPublicIP(%s) = %v, want %v", value, actual, expected)
		}
	}
}
