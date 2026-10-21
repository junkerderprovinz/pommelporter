package main

import (
	"image/color"
	"testing"
)

const square = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="#ff0000"/></svg>`

func TestRenderDrawsTheLayoutAtBothScales(t *testing.T) {
	ink, err := accentInk("#FCC419")
	if err != nil {
		t.Fatal(err)
	}
	if want := (color.NRGBA{0x8b, 0x6c, 0x0e, 0xff}); ink != want {
		t.Errorf("accent ink = %v, want %v as GlimStone derives it", ink, want)
	}
	for _, scale := range []int{1, 2} {
		img, err := render([]byte(square), "KnightLoader", ink, scale)
		if err != nil {
			t.Fatal(err)
		}
		if b := img.Bounds(); b.Dx() != width*scale || b.Dy() != height*scale {
			t.Fatalf("scale %d: size %v", scale, b.Size())
		}
		at := func(x, y int) color.NRGBA {
			return color.NRGBAModel.Convert(img.At(x*scale, y*scale)).(color.NRGBA)
		}
		for name, c := range map[string]struct {
			got, want color.NRGBA
		}{
			"ground":       {at(10, 10), ground},
			"app slot":     {at(appX, slotY), surface},
			"apps slot":    {at(appsX, slotY), surface},
			"arrow":        {at((appX+appsX)/2, slotY), ink},
			"bottom strip": {at(width/2, height-20), ground},
		} {
			if c.got != c.want {
				t.Errorf("scale %d: %s is %v, want %v", scale, name, c.got, c.want)
			}
		}
	}
}

func TestRenderRefusesALogoWithoutAViewBox(t *testing.T) {
	if _, err := render([]byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), "X", color.NRGBA{}, 1); err == nil {
		t.Error("render took a logo it cannot scale")
	}
}
