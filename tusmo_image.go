package main

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

var (
	width  = 1200
	heigth = 628

	pathToBG           = filepath.Join("assets", "images", "background.jpg")
	outputFilenameTest = "prod.png"
	fontPath           = filepath.Join("assets", "fonts", "Montserrat-Medium.ttf")
)

type TusmoPartyGameImageWaterMark struct {
	Name, URL string
}

type TusmoPartyGameImageParams struct {
	WaterMark      TusmoPartyGameImageWaterMark
	Word, Username string
}

func generateTusmoImage(params TusmoPartyGameImageParams) io.Reader {
	dc := gg.NewContext(width, heigth)

	bgImage, err := gg.LoadImage(pathToBG)
	if err != nil {
		log.Printf("An error is occured during loading background image: %v", err)
		return nil
	}

	// Fill image center
	bgImage = imaging.Fill(bgImage, dc.Width(), dc.Height(), imaging.Center, imaging.Lanczos)

	dc.DrawImage(bgImage, 0, 0)

	// Drawing
	// rgb(132 179 249)
	margin := 15.0
	x := margin
	y := margin
	w := float64(dc.Width()) - (margin * 2.0)
	h := float64(dc.Height()) - (margin * 2.0)
	dc.SetColor(color.RGBA{249, 132, 179, 215})
	dc.DrawRectangle(x, y, w, h)
	dc.Fill()

	if err := dc.LoadFontFace(fontPath, 40); err != nil {
		log.Printf("Error during load font: %v", err)
		return nil
	}

	// Watermark
	dc.SetColor(color.White)
	text := params.WaterMark.Name
	marginX := 50.0
	marginY := 15.0

	textWidth, textHeight := dc.MeasureString(text)
	x = float64(dc.Width()) - textWidth - marginX
	y = float64(dc.Height()) - textHeight - marginY
	dc.DrawString(text, x, y)

	// Print avatar image
	avatarSize := 72.0
	marginX = textWidth + marginX + 21.0
	marginY = 25.0

	avatarImage, err := getImageFromURL(params.WaterMark.URL)
	if err != nil {
		log.Printf("An error is occured during loading avatar image: %v", err)
		return nil
	}

	avatarImage = imaging.Resize(avatarImage, int(avatarSize), int(avatarSize), imaging.Lanczos)
	avatarImage = imaging.Fill(avatarImage, int(avatarSize), int(avatarSize), imaging.Center, imaging.Lanczos)

	x = float64(dc.Width()) - avatarSize - marginX
	y = float64(dc.Height()) - avatarSize - marginY

	dc.DrawImage(avatarImage, int(x), int(y))

	// Print word finded

	word := params.Word
	marginXWord := margin + 15.0
	marginYWord := margin + 221.0

	maxWidth := float64(dc.Width()) - (marginXWord * 2)

	if err := dc.LoadFontFace(fontPath, 65); err != nil {
		log.Printf("Error during load font: %v", err)
		return nil
	}

	x = marginXWord
	y = marginYWord

	// AlignCenter, x is point
	// ---------x---------
	// |                 |
	// -------------------

	dc.SetColor(color.Black)
	dc.DrawStringWrapped(word, x+1, y+1, 0, 0, maxWidth, 1.5, gg.AlignCenter)
	dc.SetColor(color.White)
	dc.DrawStringWrapped(word, x, y, 0, 0, maxWidth, 1.5, gg.AlignCenter)

	// Print name

	name := params.Username

	if err := dc.LoadFontFace(fontPath, 43); err != nil {
		log.Printf("Error during load font: %v", err)
		return nil
	}

	_, textHeight = dc.MeasureString(word)
	// 24px = 1.5 line spacing
	// y = marginYWord + ((textHeight + (24 * 3)) * float64(math.Round(textWidth/maxWidth)+1)) + 42.0
	y = marginYWord + textHeight + 42.0

	dc.SetColor(color.White)
	dc.DrawStringWrapped(name, x, y, 0, 0, maxWidth, 1.5, gg.AlignCenter)

	buff := bytes.NewBuffer([]byte{})
	png.Encode(buff, dc.Image())

	return bytes.NewReader(buff.Bytes())

	// if err := dc.SavePNG(outputFilenameTest); err != nil {
	// 	log.Fatalf("Error during save file: %v", err)
	// }
}

func getImageFromURL(url string) (image.Image, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, errors.New("Error during get content image url.")
	}
	defer res.Body.Close()

	img, _, err := image.Decode(res.Body)
	if err != nil {
		return nil, errors.New("Error during decode image retrive.")
	}

	return img, nil
}
