package imagestore

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/fyne-io/image/ico"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	_ "image/jpeg"
)

const (
	LogoMaxBytes       = 2 << 20
	FaviconMaxBytes    = 1 << 20
	BackgroundMaxBytes = 10 << 20
	LogoPrefix         = "/uploads/logos/"
	SitePrefix         = "/uploads/site/"
	BackgroundPrefix   = "/uploads/backgrounds/"
)

var ErrInvalidImage = errors.New("invalid image")

type Config struct {
	Root           string
	PublicPrefix   string
	MaxBytes       int64
	MaxDimension   int
	MaxPixels      int
	NormalizedSize int
	Formats        map[string]struct{}
}

type Store struct{ config Config }

func NewLogos(root string) *Store {
	return New(Config{
		Root: root, PublicPrefix: LogoPrefix, MaxBytes: LogoMaxBytes,
		MaxDimension: 4096, MaxPixels: 16_000_000, NormalizedSize: 512,
		Formats: formats("png", "jpeg", "webp", "ico"),
	})
}

func NewSiteLogo(root string) *Store {
	return New(Config{
		Root: root, PublicPrefix: SitePrefix, MaxBytes: LogoMaxBytes,
		MaxDimension: 4096, MaxPixels: 16_000_000, NormalizedSize: 512,
		Formats: formats("png", "jpeg", "webp", "ico"),
	})
}

func NewFavicon(root string) *Store {
	return New(Config{
		Root: root, PublicPrefix: SitePrefix, MaxBytes: FaviconMaxBytes,
		MaxDimension: 2048, MaxPixels: 4_000_000, NormalizedSize: 256,
		Formats: formats("png", "ico"),
	})
}

func NewBackgrounds(root string) *Store {
	return New(Config{
		Root: root, PublicPrefix: BackgroundPrefix, MaxBytes: BackgroundMaxBytes,
		MaxDimension: 8192, MaxPixels: 40_000_000, NormalizedSize: 2560,
		Formats: formats("png", "jpeg", "webp"),
	})
}

func New(config Config) *Store { return &Store{config: config} }

func (store *Store) Save(reader io.Reader) (string, error) {
	content, err := io.ReadAll(io.LimitReader(reader, store.config.MaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}
	if len(content) == 0 || int64(len(content)) > store.config.MaxBytes {
		return "", ErrInvalidImage
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || !store.supports(format) || !store.validDimensions(config.Width, config.Height) {
		return "", ErrInvalidImage
	}
	decoded, decodedFormat, err := image.Decode(bytes.NewReader(content))
	if err != nil || decodedFormat != format {
		return "", ErrInvalidImage
	}
	name, err := randomName()
	if err != nil {
		return "", fmt.Errorf("name image: %w", err)
	}
	temporary, err := os.CreateTemp(store.config.Root, ".image-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create image: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("protect image: %w", err)
	}
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(temporary, normalize(decoded, store.config.NormalizedSize)); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("encode image: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return "", fmt.Errorf("sync image: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close image: %w", err)
	}
	if err := os.Rename(temporaryName, filepath.Join(store.config.Root, name)); err != nil {
		return "", fmt.Errorf("store image: %w", err)
	}
	return store.config.PublicPrefix + name, nil
}

func (store *Store) Remove(publicPath string) error {
	name, ok := store.Filename(publicPath)
	if !ok {
		return nil
	}
	if err := os.Remove(filepath.Join(store.config.Root, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove image: %w", err)
	}
	return nil
}

func (store *Store) Prune(retained map[string]struct{}) error {
	entries, err := os.ReadDir(store.config.Root)
	if err != nil {
		return fmt.Errorf("read image directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		publicPath := store.config.PublicPrefix + entry.Name()
		if _, keep := retained[publicPath]; keep {
			continue
		}
		if err := store.Remove(publicPath); err != nil {
			return err
		}
	}
	return nil
}

func (store *Store) Filename(publicPath string) (string, bool) {
	return Filename(publicPath, store.config.PublicPrefix)
}

func (store *Store) Root() string    { return store.config.Root }
func (store *Store) Prefix() string  { return store.config.PublicPrefix }
func (store *Store) MaxBytes() int64 { return store.config.MaxBytes }
func (store *Store) supports(value string) bool {
	_, supported := store.config.Formats[value]
	return supported
}

func (store *Store) validDimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= store.config.MaxDimension && height <= store.config.MaxDimension && width*height <= store.config.MaxPixels
}

func Filename(publicPath, prefix string) (string, bool) {
	if !strings.HasPrefix(publicPath, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(publicPath, prefix)
	if name == "" || filepath.Base(name) != name || !strings.HasSuffix(name, ".png") || strings.ContainsAny(name, `/\`) {
		return "", false
	}
	return name, true
}

func formats(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func normalize(source image.Image, maximum int) image.Image {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= maximum && height <= maximum && bounds.Min.X == 0 && bounds.Min.Y == 0 {
		return source
	}
	if width > maximum || height > maximum {
		scale := float64(maximum) / float64(max(width, height))
		width, height = max(1, int(float64(width)*scale)), max(1, int(float64(height)*scale))
	}
	destination := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(destination, destination.Bounds(), source, bounds, draw.Over, nil)
	return destination
}

func randomName() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value) + ".png", nil
}
