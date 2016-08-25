package main

import "os"
import "log"
import "fmt"
import "time"
import "path/filepath"
import "strings"
import "unicode/utf8"
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

type event int

const (
	unknownEvent = event(iota)
	playPauseEvent
	seekNextEvent
	seekPrevEvent
	seekNextFrameEvent
	seekPrevFrameEvent
)

func main() {
	driver.Main(func(s screen.Screen) {
		// Find movie/sequence.
		//
		// TODO: User input.
		seq, err := filepath.Glob("sample/pngseq/pngseq.*.png")
		if err != nil {
			log.Fatal(err)
		}
		if seq == nil {
			log.Print("no input movie or sequence")
			os.Exit(1)
		}

		// Get initial size.
		firstImage, err := loadImage(seq[0])
		if err != nil {
			log.Fatal(err)
		}
		initSize := firstImage.Bounds().Max

		// Make a window.
		w, err := s.NewWindow(&screen.NewWindowOptions{Width: initSize.X, Height: initSize.Y})
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()

		playEventChan := make(chan event)
		playFrame := playFramer(24, len(seq)-1, w, playEventChan)

		// Keep textures so we can reuse it. (ex: play loop)
		texs := make([]screen.Texture, len(seq))

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
					playEventChan <- playPauseEvent
				}
				if e.Code == key.CodeLeftArrow && e.Direction == key.DirPress {
					playEventChan <- seekPrevEvent
				}
				if e.Code == key.CodeRightArrow && e.Direction == key.DirPress {
					playEventChan <- seekNextEvent
				}
				if e.Rune == ',' && e.Direction == key.DirPress {
					playEventChan <- seekPrevFrameEvent
				}
				if e.Rune == '.' && e.Direction == key.DirPress {
					playEventChan <- seekNextFrameEvent
				}

			case size.Event:

			case paint.Event:
				f := <-playFrame

				var tex screen.Texture
				if texs[f] == nil {
					img, err := loadImage(seq[f])
					if err != nil {
						log.Fatal(err)
					}
					tex = imageTexture(s, img)
					texs[f] = tex
				} else {
					// loop
					tex = texs[f]
				}

				subTex := subtitleTexture(s, fmt.Sprintf("play frame: %v\n\ncheck bounds", f))

				w.Copy(image.Point{}, tex, tex.Bounds(), screen.Src, nil)
				w.Copy(image.Point{0, 0}, subTex, subTex.Bounds(), screen.Over, nil)
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
		Src:  image.White,
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

// playFramer return playFrame channel that sends which frame should played at the time.
func playFramer(fps float64, endFrame int, w screen.Window, eventCh <-chan event) <-chan int {
	playFrame := make(chan int)
	go func() {
		playing := true
		start := time.Now()
		var f int
		for {
			select {
			case ev := <-eventCh:
				if playing {
					f += int(time.Since(start).Seconds() * fps)
					if f > endFrame {
						f %= endFrame
					}
				}
				start = time.Now()

				switch ev {
				case playPauseEvent:
					if playing {
						playing = false
					} else {
						playing = true
					}
				case seekPrevEvent:
					f -= int(fps) // TODO: rounding for non-integer fps
					if f < 0 {
						f = 0
					}
				case seekNextEvent:
					f += int(fps) // TODO: rounding for non-integer fps
					if f > endFrame {
						f = endFrame
					}
				case seekPrevFrameEvent:
					// when seeking frames, player should stop.
					playing = false
					f -= 1
					if f < 0 {
						f = 0
					}
				case seekNextFrameEvent:
					// when seeking frames, player should stop.
					playing = false
					f += 1
					if f > endFrame {
						f = endFrame
					}
				}
			case <-time.After(time.Second / time.Duration(fps)):
				w.Send(paint.Event{})
				var tf int
				if playing {
					tf = f + int(time.Since(start).Seconds()*fps)
					if tf > endFrame {
						tf %= endFrame
					}
				} else {
					tf = f
				}
				playFrame <- tf
			}
		}
	}()
	return playFrame
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
