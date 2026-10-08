// Package imageconv re-encodes uploaded images to WebP, matching the
// NestJS API's sharp pipeline: apply EXIF orientation, shrink to fit
// MaxDimension (never enlarge), encode at Quality, keep animation.
package imageconv

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	_ "image/jpeg" // register decoders
	_ "image/png"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
)

const (
	MaxDimension = 2000
	Quality      = 82
)

var encodeOptions = webp.Options{Quality: Quality, Method: 4}

// ToWebP converts a JPEG, PNG, GIF or WebP image to WebP.
func ToWebP(data []byte) ([]byte, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("detect image format: %w", err)
	}

	switch format {
	case "gif":
		if anim, err := gif.DecodeAll(bytes.NewReader(data)); err == nil && len(anim.Image) > 1 {
			return encodeAnimation(gifFrames(anim))
		}
	case "webp":
		if anim, err := webp.DecodeAll(bytes.NewReader(data)); err == nil && len(anim.Image) > 1 {
			return encodeAnimation(anim)
		}
	}

	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	var out bytes.Buffer
	if err := webp.Encode(&out, fit(img), encodeOptions); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return out.Bytes(), nil
}

func fit(img image.Image) image.Image {
	return imaging.Fit(img, MaxDimension, MaxDimension, imaging.Lanczos)
}

func encodeAnimation(anim *webp.WEBP) ([]byte, error) {
	if len(anim.Image) == 0 {
		return nil, errors.New("animation has no frames")
	}
	frames := make([]image.Image, len(anim.Image))
	for i, frame := range anim.Image {
		frames[i] = fit(frame)
	}
	var out bytes.Buffer
	err := webp.EncodeAll(&out, &webp.WEBP{Image: frames, Delay: anim.Delay, LoopCount: anim.LoopCount}, encodeOptions)
	if err != nil {
		return nil, fmt.Errorf("encode animated webp: %w", err)
	}
	return out.Bytes(), nil
}

// gifFrames renders each GIF frame onto the full canvas (GIF frames are
// often partial patches with disposal rules), as WebP frames are whole
// images.
func gifFrames(g *gif.GIF) *webp.WEBP {
	bounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	canvas := image.NewRGBA(bounds)
	frames := make([]image.Image, 0, len(g.Image))
	delays := make([]int, 0, len(g.Image))

	for i, frame := range g.Image {
		var previous *image.RGBA
		disposal := byte(gif.DisposalNone)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		if disposal == gif.DisposalPrevious {
			previous = image.NewRGBA(bounds)
			draw.Draw(previous, bounds, canvas, image.Point{}, draw.Src)
		}

		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		snapshot := image.NewRGBA(bounds)
		draw.Draw(snapshot, bounds, canvas, image.Point{}, draw.Src)
		frames = append(frames, snapshot)
		delays = append(delays, g.Delay[i]*10) // GIF: 1/100 s; WebP: ms

		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, frame.Bounds(), image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			draw.Draw(canvas, bounds, previous, image.Point{}, draw.Src)
		}
	}

	loop := g.LoopCount // GIF: 0 = forever, -1 = play once, n = n repeats
	if loop < 0 {
		loop = 1
	}
	return &webp.WEBP{Image: frames, Delay: delays, LoopCount: loop}
}
