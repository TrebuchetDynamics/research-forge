package webui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func solidPNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestComparePNGIdentical(t *testing.T) {
	a := solidPNG(t, 20, 20, color.RGBA{10, 20, 30, 255})
	diff, err := comparePNG(a, a)
	if err != nil {
		t.Fatalf("comparePNG: %v", err)
	}
	if !diff.SameBounds || diff.DiffRatio != 0 {
		t.Fatalf("identical images diff = %+v, want sameBounds & 0 ratio", diff)
	}
}

func TestComparePNGDifferentBounds(t *testing.T) {
	a := solidPNG(t, 20, 20, color.RGBA{0, 0, 0, 255})
	b := solidPNG(t, 21, 20, color.RGBA{0, 0, 0, 255})
	diff, err := comparePNG(a, b)
	if err != nil {
		t.Fatalf("comparePNG: %v", err)
	}
	if diff.SameBounds {
		t.Fatalf("expected SameBounds=false for differing sizes")
	}
}

func TestComparePNGToleratesSmallChannelJitter(t *testing.T) {
	a := solidPNG(t, 10, 10, color.RGBA{100, 100, 100, 255})
	// +8 per channel is within screenshotChannelThreshold (12): no pixel counts.
	b := solidPNG(t, 10, 10, color.RGBA{108, 108, 108, 255})
	diff, err := comparePNG(a, b)
	if err != nil {
		t.Fatalf("comparePNG: %v", err)
	}
	if !diff.SameBounds || diff.DiffRatio != 0 {
		t.Fatalf("small jitter diff = %+v, want 0 ratio", diff)
	}
}

func TestComparePNGCountsLargeChannelDifference(t *testing.T) {
	a := solidPNG(t, 10, 10, color.RGBA{0, 0, 0, 255})
	b := solidPNG(t, 10, 10, color.RGBA{255, 255, 255, 255})
	diff, err := comparePNG(a, b)
	if err != nil {
		t.Fatalf("comparePNG: %v", err)
	}
	if !diff.SameBounds || diff.DiffRatio != 1.0 {
		t.Fatalf("fully different images diff = %+v, want ratio 1.0", diff)
	}
}

func TestComparePNGRejectsNonImage(t *testing.T) {
	if _, err := comparePNG([]byte("not a png"), solidPNG(t, 2, 2, color.Black)); err == nil {
		t.Fatal("expected error decoding non-PNG golden")
	}
}
