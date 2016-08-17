package main

import "os"
import "log"
import "image"
import _ "image/png"
import "golang.org/x/image/draw"
import "golang.org/x/exp/shiny/driver"
import "golang.org/x/exp/shiny/screen"
import "golang.org/x/mobile/event/lifecycle"
import "golang.org/x/mobile/event/size"
import "golang.org/x/mobile/event/paint"
import "golang.org/x/mobile/event/key"
import "golang.org/x/image/font"
import "golang.org/x/image/font/inconsolata"
import "golang.org/x/image/math/fixed"

func main() {
	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(nil)
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()

		var width, height int
		var tex screen.Texture
		var subTex screen.Texture // for subtitle
		imgpath := "sample/colorbar.png"
		img, err := loadImage(imgpath)
		if err != nil {
			log.Fatal(err)
		}

		for {
			switch e := w.NextEvent().(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}

			case key.Event:
				if e.Code == key.CodeEscape {
					return
				}

			case size.Event:
				width = e.WidthPx
				height = e.HeightPx

				// Upload image texture
				t, err := s.NewTexture(image.Point{width, height})
				if err != nil {
					log.Fatal(err)
				}
				tex = t
				buf, err := s.NewBuffer(image.Point{width, height})
				if err != nil {
					tex.Release()
					log.Fatal(err)
				}
				rgba := buf.RGBA()
				draw.Copy(rgba, image.Point{}, img, img.Bounds(), draw.Src, nil)
				tex.Upload(image.Point{}, buf, rgba.Bounds())
				buf.Release()

				// Upload subtitle texture
				//
				// TODO: Fit texture size to subtitle. Could I know the size already?
				t, err = s.NewTexture(image.Point{width, height})
				if err != nil {
					log.Fatal(err)
				}
				subTex = t
				subBuf, err := s.NewBuffer(image.Point{width, height})
				subRgba := subBuf.RGBA()
				d := font.Drawer{
					Dst:  subRgba,
					Src:  image.Black,
					Face: inconsolata.Regular8x16,
					Dot: fixed.Point26_6{
						Y: inconsolata.Regular8x16.Metrics().Ascent,
					},
				}
				d.DrawString("this is a sub-title.")
				subTex.Upload(image.Point{}, subBuf, subRgba.Bounds())
				subBuf.Release()

			case paint.Event:
				w.Copy(image.Point{}, tex, image.Rect(0, 0, width, height), screen.Src, nil)
				w.Copy(image.Point{500, 500}, subTex, image.Rect(0, 0, width, height), screen.Over, nil)
				w.Publish()
			}
		}
	})
}

func loadImage(pth string) (image.Image, error) {
	f, err := os.Open(pth)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}
