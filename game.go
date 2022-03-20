package main

import (
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
	"image"
	_ "image/png"
	"os"
	"strconv"
	"strings"
	"time"
)

type anim struct { //animated sprite
	me      pixel.Sprite //the sprite associated with the struct
	index   int          //current frame the animation is on
	quantum float64      //amount of time until next frame
}

type Direction string

const ( //Direction has an X value, a Y value, and an Offset Value (controls which animation will play from spritesheet)
	S  Direction = "0,1,0"
	SW           = "1.5,0.5,1"
	W            = "2,0,2"
	NW           = "1.5,-0.5,3"
	N            = "0,-1,4"
	NE           = "-1.5,-0.5,3"
	E            = "-2, 0,2"
	SE           = "-1.5,0.5,1"
)

func run() {
	cfg := pixelgl.WindowConfig{ //set up window
		Title:  "Game",
		Bounds: pixel.R(0, 0, 1000, 1000),
		VSync:  true, //refreshes at a consistent rate
	}
	win, err := pixelgl.NewWindow(cfg) //create window
	if err != nil {
		panic(err)
	}

	runsheet, err := loadPicture("sprites/gopherrunning.png") //load spritesheets
	if err != nil {
		panic(err)
	}
	idlesheet, err := loadPicture("sprites/gopheridle.png")
	if err != nil {
		panic(err)
	}
	ringsheet, err := loadPicture("sprites/rings.png")
	if err != nil {
		panic(err)
	}

	playerSpriteScale := 84.0
	var runFrames []pixel.Rect //create Rect arrays for sprites
	for y := runsheet.Bounds().Min.Y; y < runsheet.Bounds().Max.Y; y += playerSpriteScale {
		for x := runsheet.Bounds().Min.X; x < runsheet.Bounds().Max.X; x += playerSpriteScale {
			runFrames = append(runFrames, pixel.R(x, y, x+playerSpriteScale, y+playerSpriteScale))
		}
	}
	var idleFrames []pixel.Rect
	for y := idlesheet.Bounds().Min.Y; y < idlesheet.Bounds().Max.Y; y += playerSpriteScale {
		for x := idlesheet.Bounds().Min.X; x < idlesheet.Bounds().Max.X; x += playerSpriteScale {
			idleFrames = append(idleFrames, pixel.R(x, y, x+playerSpriteScale, y+playerSpriteScale))
		}
	}
	ringScale := 34.0
	var ringFrames []pixel.Rect
	for y := ringsheet.Bounds().Min.Y; y < ringsheet.Bounds().Max.Y; y += ringScale {
		for x := ringsheet.Bounds().Min.X; x < ringsheet.Bounds().Max.X; x += ringScale {
			ringFrames = append(ringFrames, pixel.R(x, y, x+ringScale, y+ringScale))
		}
	}

	var (
		player           = anim{*pixel.NewSprite(idlesheet, idleFrames[0]), 0, 0}
		playerSpeed      = 150.0
		playerPos        = pixel.ZV
		playerDir        = S
		lastDir          = S
		playerScale      = 2.0
		playerMoving     = false
		playerAnimOffset = 0
		playerSheet      = idlesheet
		playerFrames     = idleFrames
		playerAnimSpeed  = 15
		playerFrameCount = 8

		ring = anim{*pixel.NewSprite(idlesheet, idleFrames[0]), 0, 0}

		animsList []anim
	)

	animsList = append(animsList, player)
	animsList = append(animsList, ring)

	last := time.Now()
	for !win.Closed() {
		dt := time.Since(last).Seconds() //delta time
		last = time.Now()

		if win.Pressed(pixelgl.KeyLeft) { //test against key presses and all possible key combinations
			playerMoving = true
			playerScale = 2
			if win.Pressed(pixelgl.KeyDown) {
				playerDir = SW
			} else if win.Pressed(pixelgl.KeyUp) {
				playerDir = NW
			} else {
				playerDir = W
			}
		} else if win.Pressed(pixelgl.KeyRight) {
			playerMoving = true
			playerScale = -2
			if win.Pressed(pixelgl.KeyDown) {
				playerDir = SE
			} else if win.Pressed(pixelgl.KeyUp) {
				playerDir = NE
			} else {
				playerDir = E
			}
		} else if win.Pressed(pixelgl.KeyDown) {
			playerMoving = true
			playerDir = S
		} else if win.Pressed(pixelgl.KeyUp) {
			playerMoving = true
			playerDir = N
		}
		if playerMoving { //convert direction from string to movement
			dirs := strings.Split(string(playerDir), ",")
			xVal, _ := strconv.ParseFloat(dirs[0], 32)
			yVal, _ := strconv.ParseFloat(dirs[1], 32)
			OffsetVal, _ := strconv.Atoi(dirs[2])  //get offset for animation row we want to use
			playerPos.X += xVal * playerSpeed * dt //calculate movement of character
			playerPos.Y += yVal * playerSpeed * dt
			playerAnimOffset = OffsetVal
			playerSheet = runsheet
			playerFrames = runFrames
			playerFrameCount = 12
		}
		if lastDir != playerDir { //if directions changes, reset animation
			player.index = playerAnimOffset * playerFrameCount
			player.quantum = 0
			playerFrameCount = 12
		}

		lastDir = playerDir

		if !playerMoving { //switch to idle state
			playerSheet = idlesheet
			playerFrames = idleFrames
			playerFrameCount = 8
		}

		if win.JustReleased(pixelgl.KeyLeft) || // if keys are released, player isn't moving.
			win.JustReleased(pixelgl.KeyRight) ||
			win.JustReleased(pixelgl.KeyDown) ||
			win.JustReleased(pixelgl.KeyUp) {
			playerMoving = false
		}

		win.Clear(colornames.Coral) //refresh window, set color

		//figure out the current frame of the character and draw it
		animate(&player, dt, playerAnimSpeed, playerFrameCount, playerAnimOffset, playerSheet, playerFrames)
		player.me.Draw(win, pixel.IM.ScaledXY(pixel.ZV, pixel.V(playerScale, 2)).Moved(win.Bounds().Center().Sub(playerPos)))

		animate(&ring, dt, 12, 7, 0, ringsheet, ringFrames)
		ring.me.Draw(win, pixel.IM.Scaled(pixel.ZV, 2).Moved(pixel.V(100, 100)))

		win.Update() //update window
	}
}

func loadPicture(path string) (pixel.Picture, error) {
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

func animate(subject *anim, deltaTime float64, frameRate int, numFrames int, rowOffset int, spriteSheet pixel.Picture, frames []pixel.Rect) {
	subject.quantum += deltaTime //increment quantum of whatever we're animating

	//when the quantum is equal or above the time allotted for a single frame, change to the next frame and reset
	if subject.quantum >= 1.0/float64(frameRate) {
		subject.index++     //increment to next frame of animation
		subject.quantum = 0 //reset quantum
	}

	//frameoffset allows the animation to switch to a different row of the spritesheet
	//row offset is the number row we want to use. to get there we must skip the frames that exist on the rows in between
	frameOffset := rowOffset * numFrames

	if subject.index > numFrames+frameOffset-1 {
		//index is greater than the number of frames in the animation (taking into account our offset)
		subject.index = frameOffset //reset to beginning of animation
	}
	//set sprite of our subject
	subject.me = *pixel.NewSprite(spriteSheet, frames[subject.index])
}

func main() {
	pixelgl.Run(run)
}
