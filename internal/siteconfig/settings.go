package siteconfig

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"iiiu-nav/internal/imagestore"
)

const SettingKey = "site"

type Settings struct {
	Name              string `json:"name"`
	LogoURL           string `json:"logoUrl"`
	FaviconURL        string `json:"faviconUrl"`
	AccentColor       string `json:"accentColor"`
	BackgroundURL     string `json:"backgroundUrl"`
	BackgroundOverlay int    `json:"backgroundOverlay"`
	IndexingEnabled   bool   `json:"indexingEnabled"`
}

func Default() Settings {
	return Settings{
		Name: "iiiu-nav", AccentColor: "#305880",
		BackgroundOverlay: 78, IndexingEnabled: false,
	}
}

func Validate(settings Settings) (Settings, error) {
	settings.Name = strings.TrimSpace(settings.Name)
	settings.AccentColor = strings.ToLower(strings.TrimSpace(settings.AccentColor))
	if settings.Name == "" || utf8.RuneCountInString(settings.Name) > 80 {
		return Settings{}, errors.New("site name must contain 1 to 80 characters")
	}
	if !accentPattern.MatchString(settings.AccentColor) {
		return Settings{}, errors.New("accent color must be a six-digit hexadecimal color")
	}
	if settings.BackgroundOverlay < 60 || settings.BackgroundOverlay > 95 {
		return Settings{}, errors.New("background overlay must be between 60 and 95")
	}
	if settings.LogoURL != "" {
		if _, valid := imagestore.Filename(settings.LogoURL, imagestore.SitePrefix); !valid {
			return Settings{}, errors.New("invalid site logo path")
		}
	}
	if settings.FaviconURL != "" {
		if _, valid := imagestore.Filename(settings.FaviconURL, imagestore.SitePrefix); !valid {
			return Settings{}, errors.New("invalid favicon path")
		}
	}
	if settings.BackgroundURL != "" {
		if _, valid := imagestore.Filename(settings.BackgroundURL, imagestore.BackgroundPrefix); !valid {
			return Settings{}, errors.New("invalid background path")
		}
	}
	return settings, nil
}

var accentPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)
