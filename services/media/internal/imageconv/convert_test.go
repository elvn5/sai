package imageconv_test

import (
	"bytes"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/gen2brain/webp"

	"github.com/elvn5/sai/services/media/internal/imageconv"
)

func solid(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func decodeWebP(t *testing.T, data []byte) *webp.WEBP {
	t.Helper()
	anim, err := webp.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not WebP: %v", err)
	}
	return anim
}

func TestLargeImageShrinksToFit(t *testing.T) {
	var src bytes.Buffer
	_ = png.Encode(&src, solid(4000, 1000, color.RGBA{200, 50, 50, 255}))

	out, err := imageconv.ToWebP(src.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	b := decodeWebP(t, out).Image[0].Bounds()
	if b.Dx() != 2000 || b.Dy() != 500 {
		t.Fatalf("size = %dx%d, want 2000x500", b.Dx(), b.Dy())
	}
}

func TestSmallImageIsNotEnlarged(t *testing.T) {
	var src bytes.Buffer
	_ = jpeg.Encode(&src, solid(300, 200, color.RGBA{10, 120, 200, 255}), nil)

	out, err := imageconv.ToWebP(src.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeWebP(t, out).Image[0].Bounds(); b.Dx() != 300 || b.Dy() != 200 {
		t.Fatalf("size = %dx%d, want 300x200", b.Dx(), b.Dy())
	}
}

func TestAnimatedGIFStaysAnimated(t *testing.T) {
	g := &gif.GIF{LoopCount: 0}
	for i, c := range []color.Color{color.White, color.Black, color.White} {
		frame := image.NewPaletted(image.Rect(0, 0, 40, 30), palette.Plan9)
		for y := 0; y < 30; y++ {
			for x := 0; x < 40; x++ {
				frame.Set(x, y, c)
			}
		}
		g.Image = append(g.Image, frame)
		g.Delay = append(g.Delay, 10*(i+1))
	}
	var src bytes.Buffer
	if err := gif.EncodeAll(&src, g); err != nil {
		t.Fatal(err)
	}

	out, err := imageconv.ToWebP(src.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	anim := decodeWebP(t, out)
	if len(anim.Image) != 3 {
		t.Fatalf("frames = %d, want 3", len(anim.Image))
	}
	if anim.Delay[1] != 200 {
		t.Fatalf("delay[1] = %dms, want 200ms", anim.Delay[1])
	}
}

func TestRejectsNonImage(t *testing.T) {
	if _, err := imageconv.ToWebP([]byte("definitely not an image")); err == nil {
		t.Fatal("expected an error")
	}
}
