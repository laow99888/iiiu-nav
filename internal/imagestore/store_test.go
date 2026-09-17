package imagestore

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/fyne-io/image/ico"
)

func TestLogoStoreNormalizesSupportedImagesToSafePNG(t *testing.T) {
	t.Parallel()
	for _, format := range []string{"png", "jpeg", "ico", "webp"} {
		t.Run(format, func(t *testing.T) {
			root := t.TempDir()
			store := NewLogos(root)
			imageValue := image.NewNRGBA(image.Rect(0, 0, 700, 350))
			imageValue.Set(0, 0, color.NRGBA{R: 240, A: 255})
			var input bytes.Buffer
			var err error
			switch format {
			case "png":
				err = png.Encode(&input, imageValue)
			case "jpeg":
				err = jpeg.Encode(&input, imageValue, nil)
			case "ico":
				err = ico.Encode(&input, imageValue)
			case "webp":
				content, decodeErr := base64.StdEncoding.DecodeString(testWebP)
				err = decodeErr
				_, _ = input.Write(content)
			}
			if err != nil {
				t.Fatalf("encode fixture: %v", err)
			}
			publicPath, err := store.Save(bytes.NewReader(input.Bytes()))
			if err != nil {
				t.Fatalf("save: %v", err)
			}
			name, valid := store.Filename(publicPath)
			if !valid {
				t.Fatalf("invalid generated path %q", publicPath)
			}
			file, err := os.Open(filepath.Join(root, name))
			if err != nil {
				t.Fatalf("open output: %v", err)
			}
			defer file.Close()
			config, decodedFormat, err := image.DecodeConfig(file)
			if err != nil || decodedFormat != "png" {
				t.Fatalf("decode output: format=%q err=%v", decodedFormat, err)
			}
			if config.Width > 512 || config.Height > 512 || config.Width < 1 || config.Height < 1 {
				t.Fatalf("unexpected normalized dimensions: %#v", config)
			}
			if format != "webp" && (config.Width != 512 || config.Height != 256) {
				t.Fatalf("aspect ratio was not preserved: %#v", config)
			}
		})
	}
}

func TestStoresEnforceFormatAndSizePolicies(t *testing.T) {
	t.Parallel()
	pngImage := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	var pngContent bytes.Buffer
	if err := png.Encode(&pngContent, pngImage); err != nil {
		t.Fatal(err)
	}
	var jpegContent bytes.Buffer
	if err := jpeg.Encode(&jpegContent, pngImage, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFavicon(t.TempDir()).Save(bytes.NewReader(jpegContent.Bytes())); err == nil {
		t.Fatal("favicon store accepted JPEG")
	}
	if _, err := NewBackgrounds(t.TempDir()).Save(bytes.NewReader(pngContent.Bytes())); err != nil {
		t.Fatalf("background store rejected PNG: %v", err)
	}
	if _, err := NewLogos(t.TempDir()).Save(bytes.NewReader(bytes.Repeat([]byte("x"), LogoMaxBytes+1))); err == nil {
		t.Fatal("logo store accepted oversized input")
	}
}

func TestFilenameRejectsPathTraversal(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"/uploads/logos/../secret.png", "/uploads/logos/a/b.png", "/uploads/logos/file.jpg", "C:\\secret.png"} {
		if _, valid := Filename(value, LogoPrefix); valid {
			t.Fatalf("accepted unsafe path %q", value)
		}
	}
}

func TestPruneKeepsOnlyReferencedImages(t *testing.T) {
	t.Parallel()
	store := NewLogos(t.TempDir())
	fixture := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	var content bytes.Buffer
	if err := png.Encode(&content, fixture); err != nil {
		t.Fatal(err)
	}
	keep, err := store.Save(bytes.NewReader(content.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	remove, err := store.Save(bytes.NewReader(content.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Prune(map[string]struct{}{keep: {}}); err != nil {
		t.Fatal(err)
	}
	keepName, _ := store.Filename(keep)
	removeName, _ := store.Filename(remove)
	if _, err := os.Stat(filepath.Join(store.Root(), keepName)); err != nil {
		t.Fatalf("referenced image removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), removeName)); !os.IsNotExist(err) {
		t.Fatalf("unreferenced image remains: %v", err)
	}
}

const testWebP = "UklGRrIBAABXRUJQVlA4TKUBAAAvSsAYAA8w//M///MfeJAkbXvaSG7m8Q3GfYSBJekwQztm/IcZlgwnmWImn2BK7aFmBtnVir6q//8VOkFE/xm4baTIu8c48ArEo6+B3zFKYln3pqClSCKX0begFTAXFOLXHSyF8cCNcZEG4OywuA4KVVfJCiArU7GAgJI8+lJP/OKMT/fBAjevg1cYB7YVkFuWga2lyPi5I0HFy5YTpWIHg0RZpkniRVW9odHAKOwosWuOGdxIyn2OvaCDvhg/we6TwadPBPbqBV58MsLmMJ8yZnOWk8SRz4N+QoyPL+MnamzMvcE1rHNEr91F9GKZPVUcS9w7PhhH36suB9qPeYb/oLk6cuTiJ0wOK3m5h1cKjW6EVZCYMK7dxcKCBdgP9HkKr9gkAO2P8GKZGWVdIAatQa+1IDpt6qyorVwdy01xdW8Jkfk6xjEXmVQQ+HQdFr6OKhIN34dXWq0+0qr6EJSCeeVLH9+gvGTLyqM65PQ44ihzlTXxQKjKbAvshXgir7Lil9w4L2bvMycmjQcqXaMCO6BlY28i+FOLzbfI1vEqxAhotocAAA=="

func TestBackgroundStoreKeepsOpaqueJPEGAndTransparentPNG(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store := NewBackgrounds(root)

	opaque := image.NewNRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			opaque.Set(x, y, color.NRGBA{R: 200, G: 190, B: 170, A: 255})
		}
	}
	var jpegInput bytes.Buffer
	if err := jpeg.Encode(&jpegInput, opaque, nil); err != nil {
		t.Fatalf("encode opaque jpeg fixture: %v", err)
	}
	publicPath, err := store.Save(bytes.NewReader(jpegInput.Bytes()))
	if err != nil {
		t.Fatalf("save opaque background: %v", err)
	}
	if filepath.Ext(publicPath) != ".jpg" {
		t.Fatalf("expected opaque background stored as .jpg, got %q", publicPath)
	}

	transparent := image.NewNRGBA(image.Rect(0, 0, 320, 200))
	transparent.Set(0, 0, color.NRGBA{A: 128})
	var pngInput bytes.Buffer
	if err := png.Encode(&pngInput, transparent); err != nil {
		t.Fatalf("encode transparent png fixture: %v", err)
	}
	transparentPath, err := store.Save(bytes.NewReader(pngInput.Bytes()))
	if err != nil {
		t.Fatalf("save transparent background: %v", err)
	}
	if filepath.Ext(transparentPath) != ".png" {
		t.Fatalf("expected transparent background stored as .png, got %q", transparentPath)
	}

	if _, valid := store.Filename(publicPath); !valid {
		t.Fatalf("expected .jpg background path to validate, got %q", publicPath)
	}
	retained := map[string]struct{}{transparentPath: {}}
	if err := store.Prune(retained); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, publicPath[len(BackgroundPrefix):])); !os.IsNotExist(err) {
		t.Fatalf("expected unreferenced .jpg to be pruned, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, transparentPath[len(BackgroundPrefix):])); err != nil {
		t.Fatalf("expected retained .png to survive pruning: %v", err)
	}
}
