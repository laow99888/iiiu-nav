package bookmarks

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/unicode/norm"

	"iiiu-nav/internal/navigation"
)

var (
	ErrEmptyImport        = errors.New("import file is empty")
	ErrUnsupportedFormat  = errors.New("unsupported bookmark file format")
	ErrUnsupportedCharset = errors.New("bookmark file declares an unsupported character encoding")
)

func Parse(reader io.Reader) (Batch, error) {
	content, err := readBounded(reader, MaxImportBytes)
	if err != nil {
		return Batch{}, err
	}
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf}))
	if len(trimmed) == 0 {
		return Batch{}, ErrEmptyImport
	}
	if trimmed[0] == '{' {
		return parseJSON(trimmed)
	}
	if trimmed[0] == '<' {
		return parseHTML(content)
	}
	return Batch{}, ErrUnsupportedFormat
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read bookmark import: %w", err)
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("bookmark import exceeds %d MiB", limit>>20)
	}
	return content, nil
}

type jsonDocument struct {
	Format     string     `json:"format"`
	Version    int        `json:"version"`
	Categories []Category `json:"categories"`
}

func parseJSON(content []byte) (Batch, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var document jsonDocument
	if err := decoder.Decode(&document); err != nil {
		return Batch{}, fmt.Errorf("parse application bookmark JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Batch{}, errors.New("application bookmark JSON must contain one document")
	}
	if document.Format != "iiiu-nav-bookmarks" || document.Version != 1 {
		return Batch{}, errors.New("application bookmark JSON version is not supported")
	}
	batch := Batch{Format: FormatJSON, Categories: document.Categories}
	normalizeBatch(&batch)
	return batch, nil
}

func normalizeBatch(batch *Batch) {
	for categoryIndex := range batch.Categories {
		category := &batch.Categories[categoryIndex]
		category.Name = normalizeName(category.Name)
		for linkIndex := range category.Links {
			link := &category.Links[linkIndex]
			link.Name = normalizeName(link.Name)
			link.Description = strings.TrimSpace(link.Description)
			link.URL = strings.TrimSpace(link.URL)
			if link.IconSource == "" {
				link.IconSource = navigation.IconSourceGenerated
			}
			if link.IconValue == "" && link.Name != "" {
				link.IconValue = firstRune(link.Name)
			}
			link.Problem = validateLink(*link, category.Name)
		}
	}
	merged := make([]Category, 0, len(batch.Categories))
	indexes := make(map[string]int, len(batch.Categories))
	for _, category := range batch.Categories {
		key := CategoryKey(category.Name)
		if index, exists := indexes[key]; exists {
			merged[index].Links = append(merged[index].Links, category.Links...)
			continue
		}
		indexes[key] = len(merged)
		merged = append(merged, category)
	}
	batch.Categories = merged
}

func normalizeName(value string) string {
	return norm.NFC.String(strings.TrimSpace(value))
}

func validateLink(link Link, categoryName string) string {
	switch {
	case categoryName == "" || utf8.RuneCountInString(categoryName) > 80:
		return "category_name_invalid"
	case link.Name == "" || utf8.RuneCountInString(link.Name) > 120:
		return "name_invalid"
	case utf8.RuneCountInString(link.Description) > 300:
		return "description_invalid"
	}
	parsed, err := url.Parse(link.URL)
	if err != nil || parsed.Host == "" || parsed.User != nil || (!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return "url_invalid"
	}
	if link.IconSource != navigation.IconSourceAuto && link.IconSource != navigation.IconSourceURL && link.IconSource != navigation.IconSourceUpload && link.IconSource != navigation.IconSourceGenerated {
		return "icon_invalid"
	}
	if link.IconSource == navigation.IconSourceGenerated {
		if utf8.RuneCountInString(link.IconValue) > 3 {
			return "icon_invalid"
		}
	} else if !validImportedLogoPath(link.IconValue) {
		return "icon_invalid"
	}
	return ""
}

func validImportedLogoPath(value string) bool {
	const prefix = "/uploads/logos/"
	name := strings.TrimPrefix(value, prefix)
	return name != value && name != "" && !strings.ContainsAny(name, `/\\`) && strings.HasSuffix(strings.ToLower(name), ".png")
}

func firstRune(value string) string {
	runeValue, _ := utf8.DecodeRuneInString(value)
	return string(runeValue)
}

func decodedHTML(content []byte) (io.Reader, error) {
	probe := content
	if len(probe) > 1024 {
		probe = probe[:1024]
	}
	_, label, certain := charset.DetermineEncoding(probe, "")
	if certain {
		encoding, _ := charset.Lookup(label)
		return encoding.NewDecoder().Reader(bytes.NewReader(content)), nil
	}
	if declared := declaredCharset(probe); declared != "" {
		encoding, _ := charset.Lookup(declared)
		if encoding == nil {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedCharset, declared)
		}
		return encoding.NewDecoder().Reader(bytes.NewReader(content)), nil
	}
	return bufio.NewReader(bytes.NewReader(bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf}))), nil
}

func declaredCharset(content []byte) string {
	lower := strings.ToLower(string(content))
	position := strings.Index(lower, "charset")
	if position < 0 {
		return ""
	}
	value := strings.TrimLeft(lower[position+len("charset"):], " \t\r\n=\"'")
	end := strings.IndexAny(value, " \t\r\n;\"'>/")
	if end >= 0 {
		value = value[:end]
	}
	return value
}
