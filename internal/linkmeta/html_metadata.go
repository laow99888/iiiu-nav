package linkmeta

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

func parseHTML(content []byte, pageURL *url.URL) (Result, []*url.URL, error) {
	document, err := html.Parse(strings.NewReader(string(content)))
	if err != nil {
		return Result{}, nil, err
	}
	var result Result
	var ogTitle, ogDescription string
	var icons []*url.URL
	baseURL := pageURL
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch strings.ToLower(node.Data) {
			case "base":
				if value := attribute(node, "href"); value != "" {
					if resolved, resolveErr := pageURL.Parse(value); resolveErr == nil {
						baseURL = resolved
					}
				}
			case "title":
				if result.Title == "" && node.FirstChild != nil {
					result.Title = cleanText(node.FirstChild.Data, 120)
				}
			case "meta":
				name := strings.ToLower(attribute(node, "name"))
				property := strings.ToLower(attribute(node, "property"))
				value := attribute(node, "content")
				if name == "description" && result.Description == "" {
					result.Description = cleanText(value, 300)
				}
				if property == "og:title" && ogTitle == "" {
					ogTitle = cleanText(value, 120)
				}
				if property == "og:description" && ogDescription == "" {
					ogDescription = cleanText(value, 300)
				}
			case "link":
				rel := strings.Fields(strings.ToLower(attribute(node, "rel")))
				if isIconRelation(rel) {
					if href := attribute(node, "href"); href != "" {
						if resolved, resolveErr := baseURL.Parse(href); resolveErr == nil {
							icons = append(icons, resolved)
						}
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	if result.Title == "" {
		result.Title = ogTitle
	}
	if result.Description == "" {
		result.Description = ogDescription
	}
	return result, icons, nil
}

func attribute(node *html.Node, key string) string {
	for _, item := range node.Attr {
		if strings.EqualFold(item.Key, key) {
			return strings.TrimSpace(item.Val)
		}
	}
	return ""
}

func isIconRelation(values []string) bool {
	for _, value := range values {
		if value == "icon" || value == "apple-touch-icon" || value == "mask-icon" {
			return true
		}
	}
	return false
}

func cleanText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

func deduplicateURLs(values []*url.URL) []*url.URL {
	seen := make(map[string]struct{}, len(values))
	result := make([]*url.URL, 0, len(values))
	for _, value := range values {
		if value == nil || value.Host == "" {
			continue
		}
		key := value.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}
