package siteconfig

import "testing"

func TestValidateNormalizesSiteSettings(t *testing.T) {
	t.Parallel()
	value, err := Validate(Settings{
		Name: "  My Nav  ", LogoURL: "/uploads/site/logo.png",
		FaviconURL: "/uploads/site/favicon.png", AccentColor: "#A1B2C3",
		BackgroundURL: "/uploads/backgrounds/background.png", BackgroundOverlay: 70,
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if value.Name != "My Nav" || value.AccentColor != "#a1b2c3" {
		t.Fatalf("settings were not normalized: %+v", value)
	}
}

func TestValidateRejectsUnsafeOrOutOfRangeSettings(t *testing.T) {
	t.Parallel()
	valid := Default()
	for name, mutate := range map[string]func(*Settings){
		"empty name":           func(value *Settings) { value.Name = "" },
		"invalid color":        func(value *Settings) { value.AccentColor = "red" },
		"overlay":              func(value *Settings) { value.BackgroundOverlay = 59 },
		"logo traversal":       func(value *Settings) { value.LogoURL = "/uploads/site/../logo.png" },
		"background directory": func(value *Settings) { value.BackgroundURL = "/uploads/site/background.png" },
	} {
		t.Run(name, func(t *testing.T) {
			value := valid
			mutate(&value)
			if _, err := Validate(value); err == nil {
				t.Fatal("expected invalid settings to be rejected")
			}
		})
	}
}
