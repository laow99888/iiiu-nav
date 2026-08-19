package bookmarks

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"

	"iiiu-nav/internal/navigation"
)

func parseHTML(content []byte) (Batch, error) {
	reader, err := decodedHTML(content)
	if err != nil {
		return Batch{}, err
	}
	tokenizer := html.NewTokenizer(reader)
	batch := Batch{Format: FormatHTML, Categories: make([]Category, 0)}
	categoryIndex := make(map[string]int)
	var folders []string
	var folderFrames []bool
	var pendingFolder string
	var headingText, linkText strings.Builder
	var inHeading, inLink bool
	var linkURL string
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if tokenizer.Err() != nil && !errors.Is(tokenizer.Err(), io.EOF) {
				return Batch{}, fmt.Errorf("parse bookmark HTML: %w", tokenizer.Err())
			}
			normalizeBatch(&batch)
			return batch, nil
		case html.StartTagToken:
			token := tokenizer.Token()
			switch strings.ToLower(token.Data) {
			case "h3":
				inHeading = true
				headingText.Reset()
			case "a":
				inLink = true
				linkText.Reset()
				linkURL = attributeValue(token, "href")
			case "dl":
				added := pendingFolder != ""
				if added {
					folders = append(folders, pendingFolder)
					pendingFolder = ""
				}
				folderFrames = append(folderFrames, added)
			}
		case html.TextToken:
			if inHeading {
				headingText.Write(tokenizer.Text())
			}
			if inLink {
				linkText.Write(tokenizer.Text())
			}
		case html.EndTagToken:
			token := tokenizer.Token()
			switch strings.ToLower(token.Data) {
			case "h3":
				inHeading = false
				pendingFolder = normalizeName(headingText.String())
			case "a":
				inLink = false
				categoryName := UnfiledName
				if len(folders) > 0 {
					categoryName = strings.Join(folders, " / ")
				}
				index, exists := categoryIndex[categoryName]
				if !exists {
					index = len(batch.Categories)
					categoryIndex[categoryName] = index
					batch.Categories = append(batch.Categories, Category{Name: categoryName, Links: make([]Link, 0)})
				}
				name := normalizeName(linkText.String())
				batch.Categories[index].Links = append(batch.Categories[index].Links, Link{
					Name: name, URL: strings.TrimSpace(linkURL), IconSource: navigation.IconSourceGenerated,
				})
			case "dl":
				if len(folderFrames) > 0 {
					last := len(folderFrames) - 1
					if folderFrames[last] && len(folders) > 0 {
						folders = folders[:len(folders)-1]
					}
					folderFrames = folderFrames[:last]
				}
			}
		}
	}
}

func attributeValue(token html.Token, name string) string {
	for _, attribute := range token.Attr {
		if strings.EqualFold(attribute.Key, name) {
			return attribute.Val
		}
	}
	return ""
}
