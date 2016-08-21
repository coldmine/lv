package main

import "os"
import "log"
import "fmt"
import "time"
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

		startOrStopChan := make(chan bool)
		durationChan := make(chan time.Duration)
		go playTimeChecker(w, startOrStopChan, durationChan)

		imgpath := "sample/colorbar.png"
		img, err := loadImage(imgpath)
		if err != nil {
			log.Fatal(err)
		}

		// Upload image texture
		t, err := s.NewTexture(img.Bounds().Max)
		if err != nil {
			log.Fatal(err)
		}
		tex = t
		buf, err := s.NewBuffer(img.Bounds().Max)
		if err != nil {
			tex.Release()
			log.Fatal(err)
		}
		rgba := buf.RGBA()
		draw.Copy(rgba, image.Point{}, img, img.Bounds(), draw.Src, nil)
		tex.Upload(image.Point{}, buf, rgba.Bounds())
		buf.Release()

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
				if e.Code == key.CodeSpacebar && e.Direction == key.DirPress {
					startOrStopChan <- true
				}

			case size.Event:
				width = e.WidthPx
				height = e.HeightPx

			case paint.Event:
				// Upload subtitle texture
				//
				// TODO: Fit texture size to subtitle. Could I know the size already?
				subTex, err := s.NewTexture(image.Point{width, height})
				if err != nil {
					log.Fatal(err)
				}
				subBuf, err := s.NewBuffer(image.Point{width, height})
				if err != nil {
					log.Fatal(err)
				}
				subRgba := subBuf.RGBA()
				drawer := font.Drawer{
					Dst:  subRgba,
					Src:  image.Black,
					Face: inconsolata.Regular8x16,
					Dot: fixed.Point26_6{
						Y: inconsolata.Regular8x16.Metrics().Ascent,
					},
				}
				drawer.DrawString(fmt.Sprintf("play time: %v", <-durationChan))
				subTex.Upload(image.Point{}, subBuf, subRgba.Bounds())
				subBuf.Release()

				w.Copy(image.Point{}, tex, img.Bounds(), screen.Src, nil)
				w.Copy(image.Point{500, 500}, subTex, image.Rect(0, 0, width, 100), screen.Over, nil)
				w.Publish()
			}
		}
	})
}

func playTimeChecker(w screen.Window, playOrStop <-chan bool, duration chan<- time.Duration) {
	playing := true
	startTime := time.Now()
	d := time.Duration(0)
	for {
		select {
		case <-playOrStop:
			if playing {
				playing = false
				d += time.Since(startTime)
			} else {
				playing = true
				startTime = time.Now()
			}
		case <-time.After(time.Second / 24):
			if playing {
				w.Send(paint.Event{})
				duration <- d + time.Since(startTime)
			} else {
				w.Send(paint.Event{})
				duration <- d
			}
		}
	}
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
