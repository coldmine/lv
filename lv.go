package main

import "os"
import "log"
import "fmt"
import "time"
import "strings"
import "unicode/utf8"
import "image"
import _ "image/png"
import "image/color"
import "golang.org/x/image/draw"
import "golang.org/x/exp/shiny/driver"
import "golang.org/x/exp/shiny/screen"
import "golang.org/x/mobile/event/lifecycle"
import "golang.org/x/mobile/event/size"
import "golang.org/x/mobile/event/paint"
import "golang.org/x/mobile/event/key"
import "golang.org/x/mobile/event/mouse"
import "golang.org/x/image/font"
import "golang.org/x/image/font/inconsolata"
import "golang.org/x/image/math/fixed"
import "golang.org/x/image/math/f64"

type playMode int

const (
	playRealTime = playMode(iota)
	playEveryFrame
)

func (p playMode) String() string {
	switch p {
	case playRealTime:
		return "playRealTime"
	case playEveryFrame:
		return "playEveryFrame"
	default:
		return "unknown"
	}
}

type event int

const (
	unknownEvent = event(iota)
	playPauseEvent
	seekNextEvent
	seekPrevEvent
	seekNextFrameEvent
	seekPrevFrameEvent
	playRealTimeEvent
	playEveryFrameEvent
)

// frameEvent created from playFramer and sended to window.
// So let window to know current frame is changed and what frame it is.
type frameEvent int

func main() {
	driver.Main(func(s screen.Screen) {
		// Find movie/sequence from input.
		seq := os.Args[1:]
		if len(seq) == 0 {
			// TODO: Do not exit even if there is no input.
			// We should have Open dialog.
			fmt.Fprintln(os.Stderr, "No input movie or sequence")
			fmt.Fprintln(os.Stderr, "ex) lv <mov/seq>")
			os.Exit(1)
		}
		// TODO: How could we notice whether input is movie or sequence?
		// For now, we only support sequence.

		// Get initial size.
		firstImage, err := loadImage(seq[0])
		if err != nil {
			log.Fatal(err)
		}
		initSize := firstImage.Bounds().Max
		width := initSize.X
		height := initSize.Y

		// Make a window.
		w, err := s.NewWindow(&screen.NewWindowOptions{Width: width, Height: height})
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()

		mode := playRealTime
		playEventChan := make(chan event)
		go playFramer(mode, 24, len(seq), w, playEventChan)

		// If user pressing right mouse button,
		// move mouse right will zoom the image, and left will un-zoom.
		zooming := false
		var zoomCenterX float32
		var zoomCenterY float32

		// var imageScale float32 = 1
		imageTopLeft := image.Pt(0, 0)
		imageWidth := float32(width)
		imageHeight := float32(height)
		imageRect := image.Rect(0, 0, width, height)

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
				if e.Rune == 'm' && e.Direction == key.DirPress {
					if mode == playRealTime {
						mode = playEveryFrame
						playEventChan <- playEveryFrameEvent
					} else {
						mode = playRealTime
						playEventChan <- playRealTimeEvent
					}
				}
				if e.Rune == 'f' && e.Direction == key.DirPress {
					imageRect = image.Rect(0, 0, width, height)
				}

			case mouse.Event:
				switch e.Button {
				case mouse.ButtonRight:
					if e.Direction == mouse.DirPress {
						zooming = true
						zoomCenterX = e.X
						zoomCenterY = e.Y
						imageTopLeft = imageRect.Min
						imageWidth = float32(imageRect.Dx())
						imageHeight = float32(imageRect.Dy())
					} else {
						zooming = false
					}
				}
				if zooming {
					dx := e.X - float32(zoomCenterX)
					sc := fit(dx, -100, 300, 0, 4)
					// TODO: Find good way to prevent zero scaled image.
					// Maybe we should calculate absolute scale.
					topLeftOffX := (float32(imageTopLeft.X) - zoomCenterX) * sc
					topLeftOffY := (float32(imageTopLeft.Y) - zoomCenterY) * sc
					imageRect = image.Rect(
						int(zoomCenterX+topLeftOffX),
						int(zoomCenterY+topLeftOffY),
						int(zoomCenterX+topLeftOffX+(float32(imageWidth)*sc)),
						int(zoomCenterY+topLeftOffY+(float32(imageHeight)*sc)),
					)
				}

			case size.Event:
				width, height = e.WidthPx, e.HeightPx

			case paint.Event:
				w.Publish()

			case frameEvent:
				f := int(e)
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
				subTex := subtitleTexture(s, fmt.Sprintf("play frame: %v\n\n%v", f, mode))
				playbarTex := playbarTexture(s, width, 10, f, len(seq))

				w.DrawUniform(f64.Aff3{1, 0, 0, 0, 1, 0}, color.Black, image.Rect(0, 0, width, height), screen.Src, nil)
				w.Scale(imageRect, tex, tex.Bounds(), screen.Src, nil)
				w.Copy(image.Point{0, 0}, subTex, subTex.Bounds(), screen.Over, nil)
				w.Copy(image.Point{0, height - 10}, playbarTex, playbarTex.Bounds(), screen.Src, nil)
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

func playbarTexture(s screen.Screen, width, height, frame, lenSeq int) screen.Texture {
	tex, err := s.NewTexture(image.Point{width, height})
	if err != nil {
		log.Fatal(err)
	}
	buf, err := s.NewBuffer(image.Point{width, height})
	if err != nil {
		log.Fatal(err)
	}
	rgba := buf.RGBA()

	// Draw background
	gray := color.Gray{64}
	draw.Copy(rgba, image.Point{}, image.NewUniform(gray), image.Rect(0, 0, width, height), draw.Src, nil)

	// Draw cursor
	yellow := color.RGBA{R: 255, G: 255, B: 0, A: 255}
	cs := int(float64(width) * float64(frame) / float64(lenSeq))
	cw := int(float64(width) / float64(lenSeq))
	cw++ // Integer represention of width shrinks. Draw one pixel larger always.
	draw.Copy(rgba, image.Pt(cs, 0), image.NewUniform(yellow), image.Rect(0, 0, cw, height), draw.Src, nil)

	tex.Upload(image.Point{}, buf, rgba.Bounds())
	buf.Release()

	return tex
}

// playFramer return playFrame channel that sends which frame should played at the time.
func playFramer(mode playMode, fps float64, seqLen int, w screen.Window, eventCh <-chan event) {
	playing := true
	start := time.Now()
	var f int
	for {
		select {
		case ev := <-eventCh:
			if playing {
				f += int(time.Since(start).Seconds() * fps)
				if f >= seqLen {
					f %= seqLen
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
				if f >= seqLen {
					f = seqLen - 1
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
				if f >= seqLen {
					f = seqLen - 1
				}
			case playRealTimeEvent:
				mode = playRealTime
			case playEveryFrameEvent:
				mode = playEveryFrame
			}
		case <-time.After(time.Second / time.Duration(fps)):
		}
		var tf int
		if playing {
			if mode == playRealTime {
				tf = f + int(time.Since(start).Seconds()*fps)
				if tf >= seqLen {
					tf %= seqLen
				}
			} else {
				f++
				if f >= seqLen {
					f %= seqLen
				}
				tf = f
				start = time.Now()
			}
		} else {
			tf = f
		}
		w.Send(frameEvent(tf))
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
