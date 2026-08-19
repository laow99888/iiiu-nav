package linkmeta

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

const (
	maxHTMLBytes = 1 << 20
	maxIconBytes = 2 << 20
)

var ErrUnsafeURL = errors.New("URL does not resolve to a public HTTP endpoint")

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Result struct {
	Title       string
	Description string
	Icon        []byte
	IconType    string
}

type Recognizer struct {
	client   Doer
	validate func(context.Context, *url.URL) error
}

func New() *Recognizer {
	validator := validatePublicURL
	transport := &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		ResponseHeaderTimeout: 4 * time.Second,
		TLSHandshakeTimeout:   4 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		DialContext:           publicDialer,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   7 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			return validator(request.Context(), request.URL)
		},
	}
	return &Recognizer{client: client, validate: validator}
}

// NewWithClient is intended for deterministic tests; callers still provide the
// URL policy so test servers cannot accidentally weaken production validation.
func NewWithClient(client Doer, validator func(context.Context, *url.URL) error) *Recognizer {
	return &Recognizer{client: client, validate: validator}
}

func (recognizer *Recognizer) Recognize(ctx context.Context, rawURL string) (Result, error) {
	pageURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || pageURL.Host == "" || pageURL.User != nil {
		return Result{}, ErrUnsafeURL
	}
	if err := recognizer.validate(ctx, pageURL); err != nil {
		return Result{}, err
	}

	response, err := recognizer.get(ctx, pageURL)
	if err != nil {
		return Result{}, fmt.Errorf("fetch page: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Result{}, fmt.Errorf("fetch page: status %d", response.StatusCode)
	}
	if !isHTML(response.Header.Get("Content-Type")) {
		return Result{}, errors.New("fetch page: content is not HTML")
	}
	body, err := readLimited(response.Body, maxHTMLBytes)
	if err != nil {
		return Result{}, fmt.Errorf("read page: %w", err)
	}
	body, err = decodeHTML(body, response.Header.Get("Content-Type"))
	if err != nil {
		return Result{}, fmt.Errorf("decode page: %w", err)
	}
	baseURL := response.Request.URL
	metadata, iconCandidates, err := parseHTML(body, baseURL)
	if err != nil {
		return Result{}, fmt.Errorf("parse page: %w", err)
	}
	iconCandidates = append(iconCandidates, baseURL.ResolveReference(&url.URL{Path: "/favicon.ico"}))
	for _, candidate := range deduplicateURLs(iconCandidates) {
		icon, contentType, fetchErr := recognizer.fetchIcon(ctx, candidate)
		if fetchErr == nil {
			metadata.Icon = icon
			metadata.IconType = contentType
			break
		}
	}
	return metadata, nil
}

func decodeHTML(content []byte, contentType string) ([]byte, error) {
	encoding, _, _ := charset.DetermineEncoding(content, contentType)
	reader := encoding.NewDecoder().Reader(bytes.NewReader(content))
	return io.ReadAll(reader)
}

func (recognizer *Recognizer) fetchIcon(ctx context.Context, iconURL *url.URL) ([]byte, string, error) {
	if err := recognizer.validate(ctx, iconURL); err != nil {
		return nil, "", err
	}
	response, err := recognizer.get(ctx, iconURL)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", fmt.Errorf("status %d", response.StatusCode)
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	if !strings.HasPrefix(contentType, "image/") && contentType != "application/octet-stream" && contentType != "" {
		return nil, "", errors.New("icon content type is not an image")
	}
	content, err := readLimited(response.Body, maxIconBytes)
	if err != nil {
		return nil, "", err
	}
	if len(content) == 0 {
		return nil, "", errors.New("empty icon")
	}
	return content, contentType, nil
}

func (recognizer *Recognizer) get(ctx context.Context, target *url.URL) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml,image/*;q=0.8,*/*;q=0.1")
	request.Header.Set("User-Agent", "iiiu-nav/1 metadata recognizer")
	return recognizer.client.Do(request)
}

func isHTML(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && (mediaType == "text/html" || mediaType == "application/xhtml+xml")
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, errors.New("response exceeds size limit")
	}
	return content, nil
}
