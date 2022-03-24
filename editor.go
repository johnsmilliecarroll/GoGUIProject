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
var ringimgs []*pixel.Sprite
var goblinimgs []*pixel.Sprite
var tedimgs []*pixel.Sprite
var rings []pixel.Vec
var goblins []pixel.Vec
var teds []pixel.Vec
var ringsheet1 pixel.Picture
var goblinsheet1 pixel.Picture
var tedsheet1 pixel.Picture
var ringFrames1 []pixel.Rect
var goblinFrames1 []pixel.Rect
var tedFrames1 []pixel.Rect

/*
	writes the points of a line to the text file
*/
func writeLayout(ax float64, ay float64, bx float64, by float64) {
	//create or append to file
	file, err := os.OpenFile("layout.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close() //ensure file is closed later on

	mystring := fmt.Sprintf("%f", ax) + "," +
		fmt.Sprintf("%f", ay) + "," +
		fmt.Sprintf("%f", bx) + "," +
		fmt.Sprintf("%f", by) + "," + "\n"

	_, err2 := file.WriteString(mystring)

	if err2 != nil {
		log.Fatal(err2)
	}
}

/*
	Reads in barriers that were previously created
*/
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

/*
	writes the points of a line to the text file
*/
func writeItem(tag string, x float64, y float64) {

	file, err := os.OpenFile("items.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) //create or append to file

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	mystring := fmt.Sprintf("%s", tag) + "," +
		fmt.Sprintf("%f", x) + "," +
		fmt.Sprintf("%f", y) + "," + "\n"

	_, err2 := file.WriteString(mystring)

	if err2 != nil {
		log.Fatal(err2)
	}
}

/*
	Reads in previously added items
*/
func eReadItem() {

	file, err := os.Open("items.txt") //open to read

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineElems := strings.Split(scanner.Text(), ",") //split on commas
		if len(lineElems) == 4 {
			tag := lineElems[0]
			X, _ := strconv.ParseFloat(lineElems[1], 64)
			Y, _ := strconv.ParseFloat(lineElems[2], 64)
			pos := pixel.V(X, Y)
			if tag == "ring" {
				rings = append(rings, pos)
				newimg := pixel.NewSprite(ringsheet1, ringFrames1[0])
				ringimgs = append(ringimgs, newimg)
			} else if tag == "ted" {
				teds = append(teds, pos)
				newimg := pixel.NewSprite(tedsheet1, tedFrames1[0])
				tedimgs = append(tedimgs, newimg)
			} else if tag == "goblin" {
				goblins = append(goblins, pos)
				newimg := pixel.NewSprite(goblinsheet1, goblinFrames1[0])
				goblinimgs = append(goblinimgs, newimg)
			}

		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

/*
	Main function that creates the window and runs the program
*/
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

	bgimg, err := LoadPicture("sprites/map.png") //get background image
	if err != nil {
		panic(err)
	}

	ringsheet1, err = LoadPicture("sprites/rings.png")
	if err != nil {
		panic(err)
	}

	goblinsheet1, err = LoadPicture("sprites/goblinrunning.png")
	if err != nil {
		panic(err)
	}

	tedsheet1, err = LoadPicture("sprites/tedhead.png")
	if err != nil {
		panic(err)
	}

	ringScale := 43.0
	for y := ringsheet1.Bounds().Min.Y; y < ringsheet1.Bounds().Max.Y; y += ringScale {
		for x := ringsheet1.Bounds().Min.X; x < ringsheet1.Bounds().Max.X; x += ringScale {
			ringFrames1 = append(ringFrames1, pixel.R(x, y, x+ringScale, y+ringScale))
		}
	}

	goblinScale := 152.0
	for y := goblinsheet1.Bounds().Min.Y; y < goblinsheet1.Bounds().Max.Y; y += goblinScale {
		for x := goblinsheet1.Bounds().Min.X; x < goblinsheet1.Bounds().Max.X; x += goblinScale {
			goblinFrames1 = append(goblinFrames1, pixel.R(x, y, x+goblinScale, y+goblinScale))
		}
	}

	tedScale := 65.0
	for y := tedsheet1.Bounds().Min.Y; y < tedsheet1.Bounds().Max.Y; y += tedScale {
		for x := tedsheet1.Bounds().Min.X; x < tedsheet1.Bounds().Max.X; x += tedScale {
			tedFrames1 = append(tedFrames1, pixel.R(x, y, x+tedScale, y+tedScale))
		}
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
		barrierMode     = false
		ringMode        = true
		goblinMode      = false
		tedMode         = false
		placeHolder     *pixel.Sprite //follows mouse in item placement mode
	)

	eReadLayout()
	eReadItem()

	last := time.Now()
	for !win.Closed() {
		//delta time
		dt := time.Since(last).Seconds()
		last = time.Now()
		//camera
		cam := pixel.IM.Scaled(camPos, camZoom).Moved(win.Bounds().Center().Sub(camPos))
		win.SetMatrix(cam)

		placeHolder = pixel.NewSprite(ringsheet1, ringFrames1[0]) //initialize variable

		//control camera
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

		if win.JustPressed(pixelgl.KeyTab) { //toggle between barrier and item
			barrierMode = !barrierMode
		}

		if barrierMode {

			if win.JustPressed(pixelgl.MouseButtonLeft) && placeBarrier {
				pointA = cam.Unproject(win.MousePosition())
				activePlacement = true
			}
			if win.JustPressed(pixelgl.MouseButtonLeft) && activePlacement {
				pointB = cam.Unproject(win.MousePosition())
				bar := pixel.Line{A: pointA, B: pointB}
				editorBarriers = append(editorBarriers, bar)
				if !(pointA.X == pointB.X && pointA.Y == pointB.Y) { //no stray dots
					writeLayout(pointA.X, pointA.Y, pointB.X, pointB.Y)
				}
				pointA = pointB
				activePlacement = true
				placeBarrier = false
			}
			if win.JustPressed(pixelgl.MouseButtonRight) { //right click to cancel current line
				pointA = pixel.ZV
				activePlacement = false
				placeBarrier = true
			}
		} else { //item placement mode
			if win.JustPressed(pixelgl.KeyR) { //toggle different items
				ringMode = true
				goblinMode = false
				tedMode = false
			}
			if win.JustPressed(pixelgl.KeyG) {
				goblinMode = true
				ringMode = false
				tedMode = false
			}
			if win.JustPressed(pixelgl.KeyT) {
				tedMode = true
				goblinMode = false
				ringMode = false
			}
			if ringMode {
				placeHolder = pixel.NewSprite(ringsheet1, ringFrames1[0])
				if win.JustPressed(pixelgl.MouseButtonLeft) {
					pos := cam.Unproject(win.MousePosition())
					rings = append(rings, pos)
					newimg := pixel.NewSprite(ringsheet1, ringFrames1[0])
					ringimgs = append(ringimgs, newimg)
					writeItem("ring", pos.X, pos.Y)
				}
			} else if tedMode {
				placeHolder = pixel.NewSprite(tedsheet1, tedFrames1[0])
				if win.JustPressed(pixelgl.MouseButtonLeft) {
					pos := cam.Unproject(win.MousePosition())
					teds = append(teds, pos)
					newimg := pixel.NewSprite(tedsheet1, tedFrames1[0])
					tedimgs = append(tedimgs, newimg)
					writeItem("ted", pos.X, pos.Y)
				}
			} else if goblinMode {
				placeHolder = pixel.NewSprite(goblinsheet1, goblinFrames1[0])
				if win.JustPressed(pixelgl.MouseButtonLeft) {
					pos := cam.Unproject(win.MousePosition())
					goblins = append(goblins, pos)
					newimg := pixel.NewSprite(goblinsheet1, goblinFrames1[0])
					goblinimgs = append(goblinimgs, newimg)
					writeItem("goblin", pos.X, pos.Y)
				}
			}
		}

		win.Clear(colornames.Black) //refresh window, set color
		background.Draw(win, pixel.IM.Moved(win.Bounds().Center().Sub(backgroundOffset)))

		for i := range ringimgs {
			ringimgs[i].Draw(win, pixel.IM.Moved(rings[i]))
		}

		for i := range goblinimgs {
			goblinimgs[i].Draw(win, pixel.IM.Moved(goblins[i]))
		}

		for i := range tedimgs {
			tedimgs[i].Draw(win, pixel.IM.Moved(teds[i]))
		}
		if !barrierMode {
			placeHolder.Draw(win, pixel.IM.Moved(cam.Unproject(win.MousePosition())))
		} else {
			placeHolder.Draw(win, pixel.IM)
		}

		imd := imdraw.New(nil)

		for _, line := range editorBarriers {
			imd.Color = colornames.Lime
			imd.Push(line.A)
			imd.Push(line.B)
			imd.Line(2)
		}

		if pointA != pixel.ZV && barrierMode { //ghost graphic that shows where the line will be placed
			imd.Color = colornames.White
			imd.Push(pointA, cam.Unproject(win.MousePosition()))
			imd.Line(2)
		}

		imd.Draw(win)

		win.Update() //update window
	}
}

/*
	Loads Go picture as pixel picture
*/
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
