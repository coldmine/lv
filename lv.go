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
				m := buf.RGBA()
				draw.Copy(m, image.Point{}, img, img.Bounds(), draw.Src, nil)
				d := font.Drawer{
					Dst:  m,
					Src:  image.Black,
					Face: inconsolata.Regular8x16,
					Dot: fixed.Point26_6{
						Y: inconsolata.Regular8x16.Metrics().Ascent,
					},
				}
				d.DrawString("this is sub-title.")

				tex.Upload(image.Point{}, buf, m.Bounds())
				buf.Release()

			case paint.Event:
				w.Copy(image.Point{}, tex, image.Rect(0, 0, width, height), screen.Src, nil)
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
