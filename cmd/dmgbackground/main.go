// Command dmgbackground draws the window background of an app's macOS disk
// image in the GlimStone look: the app's logo and name on top, a slot for the
// app on the left, one for Applications on the right, and an arrow between.
//
//	dmgbackground -logo logo.svg -name KnightLoader [-accent '#FCC419'] [-out dir]
//
// It writes background.png at 660x400 and background@2x.png at twice that.
// scripts/make-dmg.sh places the icons on the slots drawn here.
package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The layout in points. make-dmg.sh repeats the slot centres and the size.
// The bottom 60 pt stay plain, since Finder may hide a strip of them under
// its title bar.
const (
	width, height = 660, 400
	slotY         = 206
	appX, appsX   = 180, 480
	// A slot holds Finder's 128 pt icon and the name under it.
	slotW, slotH, slotTop = 176, 184, 126
	cardRadius            = 20 // --radius-card
	// A wide logo such as TrickWork's banner gets up to three times its
	// height in width.
	logoH, logoMaxW = 44, 132
	headerTop       = 46
	nameSize        = 22 // --text-status
)

// The light GlimStone palette. Finder shows the same background in light and
// dark mode, and its icon names are dark in the light one, the default.
var (
	ground  = color.NRGBA{0xf4, 0xf4, 0xf4, 0xff}
	surface = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	text    = color.NRGBA{0x16, 0x16, 0x16, 0xff}
)

// notoSans is Noto Sans SemiBold cut down to Latin, the face GlimStone sets
// its text in (OFL.txt).
//
//go:embed NotoSans-SemiBold.ttf
var notoSans []byte

func main() {
	logo := flag.String("logo", "", "the app's logo as SVG")
	name := flag.String("name", "", "the app's name")
	accent := flag.String("accent", "#FCC419", "the app's GlimStone accent")
	out := flag.String("out", ".", "the folder to write background.png and background@2x.png to")
	flag.Parse()
	if *logo == "" || *name == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*logo, *name, *accent, *out); err != nil {
		fmt.Fprintln(os.Stderr, "dmgbackground:", err)
		os.Exit(1)
	}
}

func run(logoPath, name, accent, out string) error {
	logo, err := os.ReadFile(logoPath)
	if err != nil {
		return err
	}
	ink, err := accentInk(accent)
	if err != nil {
		return err
	}
	for scale, file := range map[int]string{1: "background.png", 2: "background@2x.png"} {
		img, err := render(logo, name, ink, scale)
		if err != nil {
			return err
		}
		if err := writePNG(filepath.Join(out, file), img); err != nil {
			return err
		}
	}
	return nil
}

// accentInk is the accent as GlimStone uses it for a line on a light ground:
// 55% of the colour, the rest black (--accent-ink).
func accentInk(accent string) (color.NRGBA, error) {
	c, err := oksvg.ParseSVGColor(accent)
	if err != nil || c == nil {
		return color.NRGBA{}, fmt.Errorf("%q is not a colour", accent)
	}
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	mix := func(v uint8) uint8 { return uint8(float64(v)*0.55 + 0.5) }
	return color.NRGBA{mix(n.R), mix(n.G), mix(n.B), 0xff}, nil
}

func render(logo []byte, name string, ink color.NRGBA, scale int) (*image.RGBA, error) {
	s := float64(scale)
	img := image.NewRGBA(image.Rect(0, 0, width*scale, height*scale))
	draw.Draw(img, img.Bounds(), image.NewUniform(ground), image.Point{}, draw.Src)
	scanner := rasterx.NewScannerGV(img.Bounds().Dx(), img.Bounds().Dy(), img, img.Bounds())

	for _, x := range []float64{appX, appsX} {
		card := rasterx.NewFiller(img.Bounds().Dx(), img.Bounds().Dy(), scanner)
		rasterx.AddRoundRect((x-slotW/2)*s, slotTop*s, (x+slotW/2)*s, (slotTop+slotH)*s,
			cardRadius*s, cardRadius*s, 0, rasterx.RoundGap, card)
		card.SetColor(surface)
		card.Draw()
	}
	drawArrow(scanner, img.Bounds(), ink, s)

	icon, err := oksvg.ReadIconStream(bytes.NewReader(logo))
	if err != nil {
		return nil, fmt.Errorf("reading the logo: %w", err)
	}
	vw, vh := icon.ViewBox.W, icon.ViewBox.H
	if vw <= 0 || vh <= 0 {
		return nil, fmt.Errorf("the logo has no viewBox")
	}
	logoW, logoHt := logoH*vw/vh, float64(logoH)
	if logoW > logoMaxW {
		logoW, logoHt = logoMaxW, logoMaxW*vh/vw
	}
	face, err := nameFace(s)
	if err != nil {
		return nil, err
	}
	nameW := float64(font.MeasureString(face, name).Round())
	gap := 12 * s
	left := (float64(img.Bounds().Dx()) - (logoW*s + gap + nameW)) / 2
	middle := (headerTop + logoH/2) * s
	icon.SetTarget(left, middle-logoHt*s/2, logoW*s, logoHt*s)
	icon.Draw(rasterx.NewDasher(img.Bounds().Dx(), img.Bounds().Dy(), scanner), 1)

	// The name's x-height is centred on the logo, which looks centred where
	// the cap height would sit high.
	baseline := middle + float64(face.Metrics().XHeight.Round())/2
	d := font.Drawer{Dst: img, Src: image.NewUniform(text), Face: face,
		Dot: fixed.P(int(left+logoW*s+gap), int(baseline))}
	d.DrawString(name)
	return img, nil
}

// drawArrow strokes the arrow from the app's slot to the Applications slot,
// with round ends like GlimStone's line glyphs.
func drawArrow(scanner rasterx.Scanner, b image.Rectangle, ink color.NRGBA, s float64) {
	const margin, head, stroke = 18, 12, 4
	x0 := (appX + slotW/2 + margin) * s
	x1 := (appsX - slotW/2 - margin) * s
	y := slotY * s
	st := rasterx.NewStroker(b.Dx(), b.Dy(), scanner)
	st.SetStroke(fixed.Int26_6(stroke*s*64), 0, rasterx.RoundCap, rasterx.RoundCap, rasterx.RoundGap, rasterx.ArcClip)
	st.SetColor(ink)
	st.Start(rasterx.ToFixedP(x0, y))
	st.Line(rasterx.ToFixedP(x1, y))
	st.Stop(false)
	st.Start(rasterx.ToFixedP(x1-head*s, y-head*s))
	st.Line(rasterx.ToFixedP(x1, y))
	st.Line(rasterx.ToFixedP(x1-head*s, y+head*s))
	st.Stop(false)
	st.Draw()
}

func nameFace(s float64) (font.Face, error) {
	f, err := opentype.Parse(notoSans)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: nameSize * s, DPI: 72, Hinting: font.HintingNone})
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
