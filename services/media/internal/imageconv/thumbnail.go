package imageconv

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
)

// ThumbnailWidths are the widths a thumbnail can have: the site's image
// breakpoints (IMAGE_WIDTHS in apps/site/src/core/lib/image-loader.ts) —
// keep the two lists the same. A fixed set keeps the cache small.
var ThumbnailWidths = []int{128, 256, 384, 640, 828, 1080, 1280, 1920}

// ThumbnailQuality is lower than Quality: thumbnails are seen small.
const ThumbnailQuality = 75

// ErrAnimated: the image is animated; thumbnails would drop the animation,
// so the original is served instead.
var ErrAnimated = errors.New("animated image")

var thumbnailOptions = webp.Options{Quality: ThumbnailQuality, Method: 4}

// IsThumbnailWidth reports whether width is one of ThumbnailWidths.
func IsThumbnailWidth(width int) bool {
	for _, w := range ThumbnailWidths {
		if w == width {
			return true
		}
	}
	return false
}

// Thumbnail returns a WebP copy of a JPEG, PNG, GIF or WebP image at most
// width pixels wide, keeping the aspect ratio and the EXIF orientation.
// Smaller images keep their size (only re-encoded).
func Thumbnail(data []byte, width int) ([]byte, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("detect image format: %w", err)
	}
	switch format {
	case "gif":
		if anim, err := gif.DecodeAll(bytes.NewReader(data)); err == nil && len(anim.Image) > 1 {
			return nil, ErrAnimated
		}
	case "webp":
		if anim, err := webp.DecodeAll(bytes.NewReader(data)); err == nil && len(anim.Image) > 1 {
			return nil, ErrAnimated
		}
	}

	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	if img.Bounds().Dx() > width {
		img = imaging.Resize(img, width, 0, imaging.Lanczos)
	}
	var out bytes.Buffer
	if err := webp.Encode(&out, img, thumbnailOptions); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return out.Bytes(), nil
}
