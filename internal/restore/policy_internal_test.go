package restore

import "testing"

func TestAllowedArchivePathAcceptsBackgroundJPEGOnly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		directory bool
		want      bool
	}{
		{name: "uploads/backgrounds/photo.jpg", want: true},
		{name: "uploads/backgrounds/photo.png", want: true},
		{name: "uploads/logos/photo.jpg"},
		{name: "uploads/logos/photo.png", want: true},
		{name: "uploads/site/favicon.png", want: true},
		{name: "uploads/site/favicon.jpg"},
		{name: "uploads/backgrounds/photo.gif"},
		{name: "uploads/other/photo.png"},
	}
	for _, testCase := range cases {
		if got := allowedArchivePath(testCase.name, testCase.directory); got != testCase.want {
			t.Fatalf("allowedArchivePath(%q) = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}
