package main

import (
	"fmt"
	"image"
	"math"
	"os"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
)

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

type drawer interface {
	Draw(pixel.Target)
}

type colorlight struct {
	color  pixel.RGBA
	point  pixel.Vec
	angle  float64
	radius float64
	dust   float64

	spread float64

	imd *imdraw.IMDraw
}

func (cl *colorlight) apply(src, noise drawer, tmp, dst *pixelgl.Canvas) {
	// create the light arc if not created already
	if cl.imd == nil {
		imd := imdraw.New(nil)
		imd.Color(pixel.Alpha(1))
		imd.Push(0)
		imd.Color(pixel.Alpha(0))
		for angle := -cl.spread / 2; angle <= cl.spread/2; angle += cl.spread / 64 {
			imd.Push(pixel.X(1).Rotated(angle))
		}
		imd.Polygon(0)
		cl.imd = imd
	}

	// draw the light arc
	tmp.Clear(pixel.Alpha(0))
	tmp.SetMatrix(pixel.IM.Scaled(0, cl.radius).Rotated(0, cl.angle).Moved(cl.point))
	tmp.SetColorMask(pixel.Alpha(1))
	tmp.SetComposeMethod(pixel.ComposeCopy)
	cl.imd.Draw(tmp)

	// draw the noise inside the light
	tmp.SetMatrix(pixel.IM)
	tmp.SetComposeMethod(pixel.ComposeIn)
	noise.Draw(tmp)

	// draw an image inside the noisy light
	tmp.SetColorMask(cl.color)
	tmp.SetComposeMethod(pixel.ComposeIn)
	src.Draw(tmp)

	// draw the light reflected from the dust
	tmp.SetMatrix(pixel.IM.Scaled(0, cl.radius).Rotated(0, cl.angle).Moved(cl.point))
	tmp.SetColorMask(cl.color.Mul(pixel.Alpha(cl.dust)))
	tmp.SetComposeMethod(pixel.ComposeOver)
	cl.imd.Draw(tmp)

	// draw the result to the dst
	dst.SetColorMask(pixel.Alpha(1))
	dst.SetComposeMethod(pixel.ComposePlus)
	tmp.Draw(dst)
}

func run() {
	pandaPic, err := loadPicture("panda.png")
	if err != nil {
		panic(err)
	}
	noisePic, err := loadPicture("noise.png")
	if err != nil {
		panic(err)
	}

	cfg := pixelgl.WindowConfig{
		Title:  "Color Light",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	panda := pixel.NewSprite(pandaPic, pandaPic.Bounds())
	panda.SetMatrix(pixel.IM.Moved(win.Bounds().Center()))
	noise := pixel.NewSprite(noisePic, noisePic.Bounds())
	noise.SetMatrix(pixel.IM.Moved(win.Bounds().Center()))

	colors := []pixel.RGBA{
		pixel.RGB(1, 0, 0),
		pixel.RGB(0, 1, 0),
		pixel.RGB(0, 0, 1),
		pixel.RGB(1/math.Sqrt2, 1/math.Sqrt2, 0),
	}

	points := []pixel.Vec{
		pixel.V(win.Bounds().Min.X(), win.Bounds().Min.Y()),
		pixel.V(win.Bounds().Max.X(), win.Bounds().Min.Y()),
		pixel.V(win.Bounds().Max.X(), win.Bounds().Max.Y()),
		pixel.V(win.Bounds().Min.X(), win.Bounds().Max.Y()),
	}

	angles := []float64{
		math.Pi / 4,
		math.Pi/4 + math.Pi/2,
		math.Pi/4 + 2*math.Pi/2,
		math.Pi/4 + 3*math.Pi/2,
	}

	lights := make([]colorlight, 4)
	for i := range lights {
		lights[i] = colorlight{
			color:  colors[i],
			point:  points[i],
			angle:  angles[i],
			radius: 800,
			dust:   0.3,
			spread: math.Pi / math.E,
		}
	}

	speed := []float64{11.0 / 23, 13.0 / 23, 17.0 / 23, 19.0 / 23}

	tmp := pixelgl.NewCanvas(win.Bounds())
	dst := pixelgl.NewCanvas(win.Bounds())

	var (
		frames = 0
		second = time.Tick(time.Second)
		fps30  = time.Tick(time.Second / 30)
	)

	start := time.Now()
	for !win.Closed() {
		if win.Pressed(pixelgl.KeyW) {
			for i := range lights {
				lights[i].dust += 0.05
				if lights[i].dust > 1 {
					lights[i].dust = 1
				}
			}
		}
		if win.Pressed(pixelgl.KeyS) {
			for i := range lights {
				lights[i].dust -= 0.05
				if lights[i].dust < 0 {
					lights[i].dust = 0
				}
			}
		}

		since := time.Since(start).Seconds()
		for i := range lights {
			lights[i].angle = angles[i] + math.Sin(since*speed[i])*math.Pi/8
		}

		win.Clear(pixel.RGB(0, 0, 0))
		dst.Clear(pixel.Alpha(0))

		dst.SetColorMask(pixel.Alpha(0.4))
		dst.SetComposeMethod(pixel.ComposeOver)
		panda.Draw(dst)

		for i := range lights {
			lights[i].apply(panda, noise, tmp, dst)
		}

		dst.Draw(win)
		win.Update()

		<-fps30 // maintain 30 fps, because my computer couldn't handle 60 here
		frames++
		select {
		case <-second:
			win.SetTitle(fmt.Sprintf("%s | FPS: %d", cfg.Title, frames))
			frames = 0
		default:
		}
	}
}

func main() {
	pixelgl.Run(run)
}