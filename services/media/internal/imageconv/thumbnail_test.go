package imageconv_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/elvn5/sai/services/media/internal/imageconv"
)

func TestThumbnailShrinksToWidth(t *testing.T) {
	var src bytes.Buffer
	_ = png.Encode(&src, solid(2000, 1500, color.RGBA{200, 50, 50, 255}))

	out, err := imageconv.Thumbnail(src.Bytes(), 640)
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeWebP(t, out).Image[0].Bounds(); b.Dx() != 640 || b.Dy() != 480 {
		t.Fatalf("size = %dx%d, want 640x480", b.Dx(), b.Dy())
	}
}

func TestThumbnailKeepsSmallerImages(t *testing.T) {
	var src bytes.Buffer
	_ = jpeg.Encode(&src, solid(300, 200, color.RGBA{10, 120, 200, 255}), nil)

	out, err := imageconv.Thumbnail(src.Bytes(), 640)
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeWebP(t, out).Image[0].Bounds(); b.Dx() != 300 || b.Dy() != 200 {
		t.Fatalf("size = %dx%d, want 300x200", b.Dx(), b.Dy())
	}
}

func TestThumbnailRefusesAnimation(t *testing.T) {
	g := &gif.GIF{}
	for _, c := range []color.Color{color.White, color.Black} {
		frame := image.NewPaletted(image.Rect(0, 0, 20, 10), palette.Plan9)
		for y := 0; y < 10; y++ {
			for x := 0; x < 20; x++ {
				frame.Set(x, y, c)
			}
		}
		g.Image = append(g.Image, frame)
		g.Delay = append(g.Delay, 10)
	}
	var src bytes.Buffer
	if err := gif.EncodeAll(&src, g); err != nil {
		t.Fatal(err)
	}
	if _, err := imageconv.Thumbnail(src.Bytes(), 128); !errors.Is(err, imageconv.ErrAnimated) {
		t.Fatalf("err = %v, want ErrAnimated", err)
	}
}

func TestThumbnailWidths(t *testing.T) {
	if !imageconv.IsThumbnailWidth(640) || imageconv.IsThumbnailWidth(641) {
		t.Fatal("IsThumbnailWidth disagrees with ThumbnailWidths")
	}
}
