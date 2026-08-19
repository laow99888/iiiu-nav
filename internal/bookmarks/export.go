package bookmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"

	"iiiu-nav/internal/navigation"
)

type ExportScope string

const (
	ScopeAll     ExportScope = "all"
	ScopePublic  ExportScope = "public"
	ScopePrivate ExportScope = "private"
)

var ErrInvalidExportScope = errors.New("invalid bookmark export scope")

func (service *Service) ExportHTML(ctx context.Context, scope ExportScope) ([]byte, error) {
	groups, err := service.exportGroups(ctx, scope)
	if err != nil {
		return nil, err
	}
	var output strings.Builder
	output.WriteString("<!DOCTYPE NETSCAPE-Bookmark-file-1>\n")
	output.WriteString("<META HTTP-EQUIV=\"Content-Type\" CONTENT=\"text/html; charset=UTF-8\">\n")
	output.WriteString("<TITLE>iiiu-nav Bookmarks</TITLE>\n<H1>iiiu-nav Bookmarks</H1>\n<DL><p>\n")
	for _, group := range groups {
		fmt.Fprintf(&output, "  <DT><H3>%s</H3>\n  <DL><p>\n", html.EscapeString(group.Category.Name))
		for _, link := range group.Links {
			fmt.Fprintf(&output, "    <DT><A HREF=\"%s\" ADD_DATE=\"%d\">%s</A>\n",
				html.EscapeString(link.URL), link.CreatedAt.Unix(), html.EscapeString(link.Name))
			if link.Description != "" {
				fmt.Fprintf(&output, "    <DD>%s\n", html.EscapeString(link.Description))
			}
		}
		output.WriteString("  </DL><p>\n")
	}
	output.WriteString("</DL><p>\n")
	return []byte(output.String()), nil
}

func (service *Service) ExportJSON(ctx context.Context, scope ExportScope) ([]byte, error) {
	groups, err := service.exportGroups(ctx, scope)
	if err != nil {
		return nil, err
	}
	document := jsonDocument{Format: "iiiu-nav-bookmarks", Version: 1, Categories: make([]Category, 0, len(groups))}
	for _, group := range groups {
		category := Category{
			Name: group.Category.Name, IconName: group.Category.IconName,
			Visibility: group.Category.Visibility, Links: make([]Link, 0, len(group.Links)),
		}
		for _, link := range group.Links {
			category.Links = append(category.Links, Link{
				Name: link.Name, Description: link.Description, URL: link.URL,
				IconSource: link.IconSource, IconValue: link.IconValue,
			})
		}
		document.Categories = append(document.Categories, category)
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return nil, fmt.Errorf("encode application bookmark JSON: %w", err)
	}
	return output.Bytes(), nil
}

func (service *Service) exportGroups(ctx context.Context, scope ExportScope) ([]navigation.Group, error) {
	if scope != ScopeAll && scope != ScopePublic && scope != ScopePrivate {
		return nil, ErrInvalidExportScope
	}
	groups, err := service.repository.Navigation(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("read bookmark export: %w", err)
	}
	if scope == ScopeAll {
		return groups, nil
	}
	wanted := navigation.Visibility(scope)
	filtered := make([]navigation.Group, 0, len(groups))
	for _, group := range groups {
		if group.Category.Visibility == wanted {
			filtered = append(filtered, group)
		}
	}
	return filtered, nil
}
