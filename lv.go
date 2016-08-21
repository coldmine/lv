package main

import "os"
import "log"
import "fmt"
import "time"
import "strings"
import "unicode/utf8"
import "image"
import _ "image/png"
import "golang.org/x/image/draw"
import "golang.org/x/exp/shiny/driver"
import "golang.org/x/exp/shiny/screen"
import "golang.org/x/mobile/event/lifecycle"

// import "golang.org/x/mobile/event/size"
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

		playPauseSwitch := make(chan bool)
		playTime := playTimer(w, playPauseSwitch)

		imgpath := "sample/colorbar.png"
		img, err := loadImage(imgpath)
		if err != nil {
			log.Fatal(err)
		}
		tex := imageTexture(s, img)

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
					playPauseSwitch <- true
				}

			case paint.Event:
				subTex := subtitleTexture(s, fmt.Sprintf("play time: %v\n\ncheck bounds", <-playTime))

				w.Copy(image.Point{}, tex, tex.Bounds(), screen.Src, nil)
				w.Copy(image.Point{500, 500}, subTex, subTex.Bounds(), screen.Over, nil)
				w.Publish()
			}
		}
	})
}

func imageTexture(s screen.Screen, img image.Image) screen.Texture {
	tex, err := s.NewTexture(img.Bounds().Max)
	if err != nil {
		log.Fatal(err)
	}
	buf, err := s.NewBuffer(img.Bounds().Max)
	if err != nil {
		tex.Release()
		log.Fatal(err)
	}
	rgba := buf.RGBA()
	draw.Copy(rgba, image.Point{}, img, img.Bounds(), draw.Src, nil)
	tex.Upload(image.Point{}, buf, rgba.Bounds())
	buf.Release()

	return tex
}

func subtitleTexture(s screen.Screen, tx string) screen.Texture {
	lines := strings.Split(tx, "\n")
	width := 0
	for _, l := range lines {
		w := 8 * utf8.RuneCountInString(l)
		if w > width {
			width = w
		}
	}
	height := 16 * len(lines)

	tex, err := s.NewTexture(image.Point{width, height})
	if err != nil {
		log.Fatal(err)
	}
	buf, err := s.NewBuffer(image.Point{width, height})
	if err != nil {
		log.Fatal(err)
	}
	rgba := buf.RGBA()

	drawer := font.Drawer{
		Dst:  rgba,
		Src:  image.Black,
		Face: inconsolata.Regular8x16,
		Dot: fixed.Point26_6{
			Y: inconsolata.Regular8x16.Metrics().Ascent,
		},
	}
	for _, l := range lines {
		drawer.DrawString(l)
		drawer.Dot.X = 0
		drawer.Dot.Y += fixed.I(16)
	}

	tex.Upload(image.Point{}, buf, rgba.Bounds())
	buf.Release()

	return tex
}

func playTimer(w screen.Window, playPauseSwitch <-chan bool) <-chan time.Duration {
	playTime := make(chan time.Duration)
	go func() {
		playing := true
		start := time.Now()
		d := time.Duration(0)
		for {
			select {
			case <-playPauseSwitch:
				if playing {
					playing = false
					d += time.Since(start)
				} else {
					playing = true
					start = time.Now()
				}
			case <-time.After(time.Second / 24):
				if playing {
					w.Send(paint.Event{})
					playTime <- d + time.Since(start)
				} else {
					w.Send(paint.Event{})
					playTime <- d
				}
			}
		}
	}()
	return playTime
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
