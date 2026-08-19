package bookmarks

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	"golang.org/x/text/cases"

	"iiiu-nav/internal/navigation"
)

type Service struct {
	repository Repository
}

func New(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Preview(ctx context.Context, reader io.Reader, visibility navigation.Visibility) (Preview, error) {
	if !validVisibility(visibility) {
		return Preview{}, errorsInvalidOptions()
	}
	batch, err := Parse(reader)
	if err != nil {
		return Preview{}, err
	}
	groups, err := service.repository.Navigation(ctx, true)
	if err != nil {
		return Preview{}, fmt.Errorf("read import comparison data: %w", err)
	}
	return buildPreview(batch, groups, visibility), nil
}

func (service *Service) Commit(ctx context.Context, reader io.Reader, options ImportOptions) (CommitResult, error) {
	if !validVisibility(options.Visibility) || !validStrategy(options.Duplicates) {
		return CommitResult{}, errorsInvalidOptions()
	}
	batch, err := Parse(reader)
	if err != nil {
		return CommitResult{}, err
	}
	return service.repository.ImportBookmarks(ctx, batch, options)
}

func buildPreview(batch Batch, groups []navigation.Group, visibility navigation.Visibility) Preview {
	preview := Preview{Format: batch.Format, Visibility: visibility, Categories: make([]PreviewCategory, 0, len(batch.Categories))}
	existingCategories := make(map[string]navigation.Category, len(groups))
	knownURLs := make(map[string]string)
	for _, group := range groups {
		existingCategories[CategoryKey(group.Category.Name)] = group.Category
		for _, link := range group.Links {
			knownURLs[CanonicalURL(link.URL)] = link.Name
		}
	}
	for _, category := range batch.Categories {
		item := PreviewCategory{Name: category.Name, Mapping: MappingNew, Links: make([]PreviewLink, 0, len(category.Links))}
		if existing, found := existingCategories[CategoryKey(category.Name)]; found {
			item.Mapping = MappingExisting
			item.ExistingID = fmt.Sprint(existing.ID)
			preview.Summary.ExistingCategories++
		} else {
			preview.Summary.NewCategories++
		}
		for _, link := range category.Links {
			entry := PreviewLink{Name: link.Name, URL: link.URL, Status: StatusNew, Problem: link.Problem}
			if link.Problem != "" {
				entry.Status = StatusInvalid
				preview.Summary.Invalid++
			} else if duplicateName, found := knownURLs[CanonicalURL(link.URL)]; found {
				entry.Status = StatusDuplicate
				entry.DuplicateOf = duplicateName
				preview.Summary.Duplicates++
			} else {
				knownURLs[CanonicalURL(link.URL)] = link.Name
				preview.Summary.New++
			}
			item.Links = append(item.Links, entry)
		}
		preview.Categories = append(preview.Categories, item)
	}
	return preview
}

func CanonicalURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return rawURL
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	hostname := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		parsed.Host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		parsed.Host = "[" + hostname + "]"
	} else {
		parsed.Host = hostname
	}
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String()
}

func CategoryKey(name string) string { return cases.Fold().String(normalizeName(name)) }
func validVisibility(value navigation.Visibility) bool {
	return value == navigation.VisibilityPublic || value == navigation.VisibilityPrivate
}
func validStrategy(value DuplicateStrategy) bool {
	return value == DuplicateSkip || value == DuplicateUpdate || value == DuplicateCreate
}
func errorsInvalidOptions() error { return fmt.Errorf("invalid bookmark import options") }
