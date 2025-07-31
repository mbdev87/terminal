package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type TextCanvas struct {
	lines      []string
	lock       sync.RWMutex
	img        *canvas.Image
	viewportH  int
	scrollOff  int
	lineHeight int
	fontFace   font.Face
	width      int
	height     int
	updateCh   chan struct{}
}

func NewTextCanvas(width, height int) *TextCanvas {
	tc := &TextCanvas{
		lines:      make([]string, 0, 1000),
		fontFace:   basicfont.Face7x13,
		width:      width,
		height:     height,
		lineHeight: 14,
		updateCh:   make(chan struct{}, 1),
	}

	tc.img = canvas.NewImageFromImage(tc.render())
	tc.img.FillMode = canvas.ImageFillContain
	tc.img.SetMinSize(fyne.NewSize(float32(width), float32(height)))

	go tc.updateLoop()

	return tc
}

func (tc *TextCanvas) updateLoop() {
	ticker := time.NewTicker(1 * time.Second / 60) // fps cap
	for range ticker.C {
		select {
		case <-tc.updateCh:
			tc.lock.RLock()
			img := tc.render()
			tc.lock.RUnlock()
			tc.img.Image = img
			fyne.Do(func() {
				canvas.Refresh(tc.img)
			})
		default:
			// no updates pending
		}
	}
}

func (tc *TextCanvas) render() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, tc.width, tc.height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{200, 200, 200, 255}),
		Face: tc.fontFace,
		Dot:  fixed.P(2, tc.lineHeight),
	}

	totalLines := len(tc.lines)
	linesPerPage := tc.height / tc.lineHeight

	// clamp offset
	if tc.scrollOff > totalLines-linesPerPage {
		tc.scrollOff = totalLines - linesPerPage
	}
	if tc.scrollOff < 0 {
		tc.scrollOff = 0
	}

	// safe slicin'
	start := tc.scrollOff
	end := start + linesPerPage
	if end > totalLines {
		end = totalLines
	}

	for _, line := range tc.lines[start:end] {
		d.Dot.X = fixed.I(2) // caret return for each new line so we don't drift
		d.DrawString(line)
		d.Dot.Y += fixed.I(tc.lineHeight)
	}

	return img
}

func (tc *TextCanvas) Append(line string) {
	tc.lock.Lock()
	defer tc.lock.Unlock()
	tc.lines = append(tc.lines, line)
	if len(tc.lines) > 10000 {
		tc.lines = tc.lines[len(tc.lines)-10000:]
	}
	tc.scrollOff = len(tc.lines) - tc.height/tc.lineHeight
	tc.requestUpdate()
}

func (tc *TextCanvas) Scroll(offset int) {
	tc.lock.Lock()
	defer tc.lock.Unlock()
	tc.scrollOff += offset
	if tc.scrollOff < 0 {
		tc.scrollOff = 0
	}
	maxOff := len(tc.lines) - tc.height/tc.lineHeight
	if tc.scrollOff > maxOff {
		tc.scrollOff = maxOff
	}
	tc.requestUpdate()
}

func (tc *TextCanvas) requestUpdate() {
	select {
	case tc.updateCh <- struct{}{}:
	default:
		// scheduled already
	}
}

func (tc *TextCanvas) Widget() fyne.CanvasObject {
	return tc.img
}

func main() {
	a := app.New()
	w := a.NewWindow("TextCanvas Demo")

	textCanvas := NewTextCanvas(800, 600)

	go func() {
		i := 0
		for {
			time.Sleep(1 * time.Millisecond)
			textCanvas.Append(fmt.Sprintf("Much canvas %04d: so wow", i))
			i++
			if i == 1000 {
				break
			}
		}
	}()

	w.SetContent(container.NewScroll(textCanvas.Widget()))
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}
