package main

import (
	"bufio"
	"fmt"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
	"image"
	_ "image/png"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

var editorBarriers []pixel.Line

func writeLayout(ax float64, ay float64, bx float64, by float64) {

	file, err := os.OpenFile("layout.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //create or append to file

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	mystring := fmt.Sprintf("%f", ax) + "," +
		fmt.Sprintf("%f", ay) + "," +
		fmt.Sprintf("%f", bx) + "," +
		fmt.Sprintf("%f", by) + "," + "\n"

	_, err2 := file.WriteString(mystring)

	if err2 != nil {
		log.Fatal(err2)
	}
}

func eReadLayout() {

	file, err := os.Open("layout.txt") //open to read

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineElems := strings.Split(scanner.Text(), ",") //split on commas
		if len(lineElems) == 5 {
			pointAX, _ := strconv.ParseFloat(lineElems[0], 64)
			pointAY, _ := strconv.ParseFloat(lineElems[1], 64)
			pointBX, _ := strconv.ParseFloat(lineElems[2], 64)
			pointBY, _ := strconv.ParseFloat(lineElems[3], 64)
			newbar := pixel.Line{A: pixel.V(pointAX, pointAY), B: pixel.V(pointBX, pointBY)}
			editorBarriers = append(editorBarriers, newbar)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func runEditor() {
	cfg := pixelgl.WindowConfig{ //set up window
		Title:  "Game",
		Bounds: pixel.R(0, 0, 1300, 1000),
		VSync:  true, //refreshes at a consistent rate
	}
	win, err := pixelgl.NewWindow(cfg) //create window
	if err != nil {
		panic(err)
	}

	bgimg, err := LoadPicture("sprites/map.png")
	if err != nil {
		panic(err)
	}

	var (
		background       = pixel.NewSprite(bgimg, bgimg.Bounds())
		backgroundOffset = pixel.V(-80, -520)

		camPos       = pixel.ZV
		camSpeed     = 500.0
		camZoom      = 1.0
		camZoomSpeed = 1.2

		pointA          = pixel.ZV
		pointB          = pixel.ZV
		placeBarrier    = true
		activePlacement = false
		barrierMode     = true
		contiguous      = true
	)

	if barrierMode {
		eReadLayout()
	}

	last := time.Now()
	for !win.Closed() {
		dt := time.Since(last).Seconds()
		last = time.Now()

		cam := pixel.IM.Scaled(camPos, camZoom).Moved(win.Bounds().Center().Sub(camPos))
		win.SetMatrix(cam)

		if win.Pressed(pixelgl.KeyLeft) || win.Pressed(pixelgl.KeyA) {
			camPos.X -= camSpeed * dt
		}
		if win.Pressed(pixelgl.KeyRight) || win.Pressed(pixelgl.KeyD) {
			camPos.X += camSpeed * dt
		}
		if win.Pressed(pixelgl.KeyDown) || win.Pressed(pixelgl.KeyS) {
			camPos.Y -= camSpeed * dt
		}
		if win.Pressed(pixelgl.KeyUp) || win.Pressed(pixelgl.KeyW) {
			camPos.Y += camSpeed * dt
		}
		camZoom *= math.Pow(camZoomSpeed, win.MouseScroll().Y)

		if barrierMode {
			if win.JustPressed(pixelgl.MouseButtonLeft) && placeBarrier {
				pointA = cam.Unproject(win.MousePosition())
				activePlacement = true
			}
			if win.JustPressed(pixelgl.MouseButtonLeft) && activePlacement {
				pointB = cam.Unproject(win.MousePosition())
				bar := pixel.Line{A: pointA, B: pointB}
				editorBarriers = append(editorBarriers, bar)
				writeLayout(pointA.X, pointA.Y, pointB.X, pointB.Y)
				if contiguous {
					pointA = pointB
					activePlacement = true
					placeBarrier = false
				} else {
					pointA = pixel.ZV
					activePlacement = false
					placeBarrier = true
				}
			}
			if win.JustPressed(pixelgl.MouseButtonRight) {
				pointA = pixel.ZV
				activePlacement = false
				placeBarrier = true
			}
		}

		win.Clear(colornames.Black) //refresh window, set color
		background.Draw(win, pixel.IM.Scaled(pixel.ZV, 1).Moved(win.Bounds().Center().Sub(backgroundOffset)))

		imd := imdraw.New(nil)

		for _, line := range editorBarriers {
			imd.Color = colornames.Lime
			imd.Push(line.A)
			imd.Push(line.B)
			imd.Line(2)
		}

		if pointA != pixel.ZV { //ghost graphic that shows where the line will be placed
			imd.Color = colornames.White
			imd.Push(pointA, cam.Unproject(win.MousePosition()))
			imd.Line(2)
		}

		imd.Draw(win)

		win.Update() //update window
	}
}

func LoadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			print("failed to load sprite with the path " + path)
		}
	}(file)
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

func main() {
	pixelgl.Run(runEditor)
}
