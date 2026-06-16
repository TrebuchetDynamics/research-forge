package webui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png" // register PNG decoder
)

// screenshotChannelThreshold is the per-channel (8-bit) tolerance used when
// comparing screenshots, absorbing minor anti-aliasing jitter between renders.
const screenshotChannelThreshold = 12

// ScreenshotDiff summarizes a pixel comparison between two PNG screenshots.
type ScreenshotDiff struct {
	// SameBounds is false when the images differ in size (always a regression).
	SameBounds bool
	// DiffRatio is the fraction of pixels (0..1) that differ by more than
	// screenshotChannelThreshold on any channel. Only meaningful when SameBounds.
	DiffRatio float64
}

// comparePNG decodes two PNG images and reports how much they differ. It is the
// deterministic core of the screenshot regression test and is unit-tested
// without a browser.
func comparePNG(golden, actual []byte) (ScreenshotDiff, error) {
	g, _, err := image.Decode(bytes.NewReader(golden))
	if err != nil {
		return ScreenshotDiff{}, fmt.Errorf("decode golden: %w", err)
	}
	a, _, err := image.Decode(bytes.NewReader(actual))
	if err != nil {
		return ScreenshotDiff{}, fmt.Errorf("decode actual: %w", err)
	}
	gb, ab := g.Bounds(), a.Bounds()
	if gb.Dx() != ab.Dx() || gb.Dy() != ab.Dy() {
		return ScreenshotDiff{SameBounds: false}, nil
	}
	total := gb.Dx() * gb.Dy()
	if total == 0 {
		return ScreenshotDiff{SameBounds: true}, nil
	}
	diff := 0
	for y := 0; y < gb.Dy(); y++ {
		for x := 0; x < gb.Dx(); x++ {
			if pixelDiffers(g.At(gb.Min.X+x, gb.Min.Y+y), a.At(ab.Min.X+x, ab.Min.Y+y)) {
				diff++
			}
		}
	}
	return ScreenshotDiff{SameBounds: true, DiffRatio: float64(diff) / float64(total)}, nil
}

func pixelDiffers(p, q color.Color) bool {
	pr, pg, pb, pa := p.RGBA()
	qr, qg, qb, qa := q.RGBA()
	return channelDiffers(pr, qr) || channelDiffers(pg, qg) || channelDiffers(pb, qb) || channelDiffers(pa, qa)
}

func channelDiffers(a, b uint32) bool {
	// RGBA() returns 16-bit pre-multiplied values; compare in 8-bit space.
	a8 := int(a >> 8)
	b8 := int(b >> 8)
	d := a8 - b8
	if d < 0 {
		d = -d
	}
	return d > screenshotChannelThreshold
}
