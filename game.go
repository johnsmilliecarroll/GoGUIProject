package main

import (
	"bufio"
	"fmt"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
	"image"
	_ "image/png"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type circle struct {
	center pixel.Vec
	radius float64
}

type line struct {
	A pixel.Vec
	B pixel.Vec
}

type anim struct { //animated entity
	me        pixel.Sprite //the sprite associated with the struct
	tag       string
	index     int     //current frame the animation is on
	quantum   float64 //amount of time until next frame
	col       circle  //collider
	pos       pixel.Vec
	scale     pixel.Vec
	dir       Direction
	speed     float64
	sortLayer int
}

type Direction string

const ( //Direction has an X value, a Y value, and an Offset Value (controls which animation will play from spritesheet)
	S  Direction = "0,1,0"
	SW           = "1.3,0.6,1"
	W            = "2,0,2"
	NW           = "1.3,-0.6,3"
	N            = "0,-1,4"
	NE           = "-1.3,-0.6,3"
	E            = "-2, 0,2"
	SE           = "-1.3,0.6,1"
)

/*
	Takes a direction and rotates clockwise
*/
func nextDir(current Direction) Direction {
	newDir := S
	if current == S {
		newDir = SW
	} else if current == SW {
		newDir = W
	} else if current == W {
		newDir = NW
	} else if current == NW {
		newDir = N
	} else if current == N {
		newDir = NE
	} else if current == NE {
		newDir = E
	} else if current == E {
		newDir = SE
	} else if current == SE {
		newDir = S
	}
	return newDir
}

type goblinKnowledge struct { //stores stuff a goblin needs to know
	LastDir   Direction //goblin needs to keep track of his last direction
	follow    bool      //if goblin is following or not
	offset    int       //what goblin's animation offset should be
	timeSpent float64   //seconds since goblin has last collided
}

var barriers []line //all collider barriers to (hopefully) keep entities from leaving the play space

var animsList []anim //list of all animated characters/entities

var score = 0 //number of rings the player has collected

var ringsheet pixel.Picture
var goblinsheet pixel.Picture
var tedsheet pixel.Picture
var ringFrames []pixel.Rect
var goblinFrames []pixel.Rect
var tedFrames []pixel.Rect

var goblinfo []goblinKnowledge //the brains of our goblins

/*
	Basically what would normally be our main, reworked for pixel. Called in the main function.
	Creates a window and all the things within it.
*/
func run() {
	cfg := pixelgl.WindowConfig{ //set up window
		Title:  "Game",
		Bounds: pixel.R(0, 0, 1300, 1000),
		VSync:  true, //refreshes at a consistent rate
	}
	win, err := pixelgl.NewWindow(cfg) //create window
	if err != nil {
		panic(err)
	}

	//region Load our images
	runsheet, err := loadPicture("sprites/gopherrunning.png") //load spritesheets
	if err != nil {
		panic(err)
	}
	idlesheet, err := loadPicture("sprites/gopheridle.png")
	if err != nil {
		panic(err)
	}
	ringsheet, err = loadPicture("sprites/rings.png")
	if err != nil {
		panic(err)
	}
	goblinsheet, err = loadPicture("sprites/goblinrunning.png")
	if err != nil {
		panic(err)
	}
	tedsheet, err = loadPicture("sprites/tedhead.png")
	if err != nil {
		panic(err)
	}
	bgimg, err := loadPicture("sprites/map.png")
	if err != nil {
		panic(err)
	}
	bgimg2, err := loadPicture("sprites/mapoverlay.png")
	if err != nil {
		panic(err)
	}
	//endregion

	//region Load all our sprite sheets
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
	ringScale := 43.0
	for y := ringsheet.Bounds().Min.Y; y < ringsheet.Bounds().Max.Y; y += ringScale {
		for x := ringsheet.Bounds().Min.X; x < ringsheet.Bounds().Max.X; x += ringScale {
			ringFrames = append(ringFrames, pixel.R(x, y, x+ringScale, y+ringScale))
		}
	}
	goblinScale := 152.0
	for y := goblinsheet.Bounds().Min.Y; y < goblinsheet.Bounds().Max.Y; y += goblinScale {
		for x := goblinsheet.Bounds().Min.X; x < goblinsheet.Bounds().Max.X; x += goblinScale {
			goblinFrames = append(goblinFrames, pixel.R(x, y, x+goblinScale, y+goblinScale))
		}
	}
	tedScale := 65.0
	for y := tedsheet.Bounds().Min.Y; y < tedsheet.Bounds().Max.Y; y += tedScale {
		for x := tedsheet.Bounds().Min.X; x < tedsheet.Bounds().Max.X; x += tedScale {
			tedFrames = append(tedFrames, pixel.R(x, y, x+tedScale, y+tedScale))
		}
	}
	//endregion

	ReadLayout() //load in level barriers from text file
	ReadItems()  //load in items from text file

	var (
		player = anim{*pixel.NewSprite(idlesheet, idleFrames[0]), "player", 0, 0,
			circle{pixel.ZV, 15}, pixel.ZV, pixel.V(1, 1), S, 150, 0}
		lastDir          = S
		playerMoving     = false
		playerAnimOffset = 0
		playerSheet      = idlesheet
		playerFrames     = idleFrames
		playerAnimSpeed  = 15
		playerFrameCount = 8

		ringicon = anim{*pixel.NewSprite(ringsheet, ringFrames[0]), "ring", 0, 0,
			circle{pixel.ZV, 10}, pixel.V(500, 300), pixel.V(1, 1),
			S, 0, 300}

		background       = pixel.NewSprite(bgimg, bgimg.Bounds())
		bgOverlay        = pixel.NewSprite(bgimg2, bgimg2.Bounds())
		backgroundOffset = pixel.V(-80, -520)

		camZoom = 2.0
		frames  = 0
		second  = time.Tick(time.Second)

		DEBUG = false
	)
	animsList = append(animsList, player)

	last := time.Now() //main game loop
	for !win.Closed() {
		dt := time.Since(last).Seconds() //delta time
		last = time.Now()

		//region PLAYER MOVEMENT
		if win.Pressed(pixelgl.KeyLeft) || win.Pressed(pixelgl.KeyA) { //test against key presses and all possible key combinations
			playerMoving = true
			player.scale = pixel.V(1, 1) //face left
			if win.Pressed(pixelgl.KeyDown) || win.Pressed(pixelgl.KeyS) {
				player.dir = SW //set player look direction
			} else if win.Pressed(pixelgl.KeyUp) || win.Pressed(pixelgl.KeyW) {
				player.dir = NW
			} else {
				player.dir = W
			}
		} else if win.Pressed(pixelgl.KeyRight) || win.Pressed(pixelgl.KeyD) {
			playerMoving = true
			player.scale = pixel.V(-1, 1) //flip image to face right
			if win.Pressed(pixelgl.KeyDown) || win.Pressed(pixelgl.KeyS) {
				player.dir = SE
			} else if win.Pressed(pixelgl.KeyUp) || win.Pressed(pixelgl.KeyW) {
				player.dir = NE
			} else {
				player.dir = E
			}
		} else if win.Pressed(pixelgl.KeyDown) || win.Pressed(pixelgl.KeyS) {
			playerMoving = true
			player.dir = S
		} else if win.Pressed(pixelgl.KeyUp) || win.Pressed(pixelgl.KeyW) {
			playerMoving = true
			player.dir = N
		}
		if playerMoving { //convert direction from string to movement
			dirs := strings.Split(string(player.dir), ",")
			xVal, _ := strconv.ParseFloat(dirs[0], 32)
			yVal, _ := strconv.ParseFloat(dirs[1], 32)
			OffsetVal, _ := strconv.Atoi(dirs[2])    //get offset for animation row we want to use
			player.pos.X += xVal * player.speed * dt //calculate movement of character
			player.pos.Y += yVal * player.speed * dt
			playerAnimOffset = OffsetVal
			playerSheet = runsheet
			playerFrames = runFrames
			playerFrameCount = 12
		}
		if lastDir != player.dir { //if directions changes, reset animation
			player.index = playerAnimOffset * playerFrameCount
			player.quantum = 0
			playerFrameCount = 12
		}

		lastDir = player.dir

		if !playerMoving { //switch to idle state
			playerSheet = idlesheet
			playerFrames = idleFrames
			playerFrameCount = 8
		}

		// if keys are released, player isn't moving.
		if win.JustReleased(pixelgl.KeyLeft) || win.JustReleased(pixelgl.KeyA) ||
			win.JustReleased(pixelgl.KeyRight) || win.JustReleased(pixelgl.KeyD) ||
			win.JustReleased(pixelgl.KeyDown) || win.JustReleased(pixelgl.KeyS) ||
			win.JustReleased(pixelgl.KeyUp) || win.JustReleased(pixelgl.KeyW) {
			playerMoving = false
		}
		//endregion

		if win.JustPressed(pixelgl.KeyR) { //R to respawn at the beginning
			player.pos = pixel.V(0, 0)
		}

		//camera
		cam := pixel.IM.Scaled(win.Bounds().Center().Sub(player.pos), camZoom).Moved(player.pos)
		win.SetMatrix(cam)

		win.Clear(colornames.Black) //refresh window, set color
		background.Draw(win, pixel.IM.Scaled(pixel.ZV, 1).Moved(win.Bounds().Center().Sub(backgroundOffset)))

		//figure out the current frame of the character and draw it
		animate(&player, dt, playerAnimSpeed, playerFrameCount, playerAnimOffset, playerSheet, playerFrames)
		playerTruePos := win.Bounds().Center().Sub(player.pos)
		//player.me.Draw(win, pixel.IM.ScaledXY(pixel.ZV, player.scale).Moved(win.Bounds().Center().Sub(player.pos)))
		player.sortLayer = int(playerTruePos.Y - 35) //offset so its at the bottom of the character
		player.col.center = pixel.V(playerTruePos.X, playerTruePos.Y-20)
		if checkCollision(&player) > 0 { //check Collision returns number of collisions taking place. if more than 1, slow down player
			player.speed = 75
		} else {
			player.speed = 150
		}
		animCollisions(&player) //check collisions against other anims

		//sort sprites to be drawn according to sorting layer.
		sort.Slice(animsList, func(j, i int) bool {
			return animsList[i].sortLayer < animsList[j].sortLayer
		})

		for i := 0; i < len(animsList); i++ {
			if animsList[i].tag == "player" {
				animsList[i].sortLayer = player.sortLayer
				player.me.Draw(win, pixel.IM.ScaledXY(pixel.ZV, player.scale).Moved(win.Bounds().Center().Sub(player.pos)))
			} else if animsList[i].tag == "ring" {
				animate(&animsList[i], dt, 12, 7, 0, ringsheet, ringFrames)
				animsList[i].me.Draw(win, pixel.IM.Scaled(pixel.ZV, 1).Moved(animsList[i].pos))
				animsList[i].col.center = animsList[i].pos
			} else if animsList[i].tag == "ted" {
				animate(&animsList[i], dt, 12, 7, 0, tedsheet, tedFrames)
				animsList[i].me.Draw(win, pixel.IM.Scaled(pixel.ZV, 1).Moved(animsList[i].pos))
				animsList[i].col.center = pixel.V(animsList[i].pos.X, animsList[i].pos.Y-50)
			} else { //it must be a goblin
				infoindex, _ := strconv.Atoi(animsList[i].tag) //goblin's tag is it's goblinfo index
				//call movement code for goblin, returns goblinKnowledge for that goblin
				goblinMovement(&animsList[i], dt, playerTruePos, &goblinfo[infoindex]) //move goblin, update goblin's knowledge
				animate(&animsList[i], dt, 12, 8, goblinfo[infoindex].offset, goblinsheet, goblinFrames)
				animsList[i].me.Draw(win, pixel.IM.ScaledXY(pixel.ZV, animsList[i].scale).Moved(animsList[i].pos))
			}
		}

		bgOverlay.Draw(win, pixel.IM.Scaled(pixel.ZV, 1).Moved(win.Bounds().Center().Sub(backgroundOffset)))

		if win.JustPressed(pixelgl.KeyTab) {
			DEBUG = !DEBUG //toggle debug mode
		}
		//draw barriers (DEBUG)
		if DEBUG {
			imd := imdraw.New(nil)
			imd.Color = colornames.Blue
			imd.Push(player.col.center)
			imd.Circle(player.col.radius, 2)
			for i := 0; i < len(animsList); i++ {
				imd.Color = colornames.Cyan
				imd.Push(animsList[i].col.center)
				imd.Circle(animsList[i].col.radius, 2)
			}
			for _, line := range barriers {
				imd.Color = colornames.Lime
				imd.Push(line.A) //draw a line with 2 points
				imd.Push(line.B)
				imd.Line(2)
			}

			imd.Draw(win) //draw debug graphics
		}

		//draw score stuff at top of screen
		txtAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
		scoreText := text.New(pixel.V(470, 350), txtAtlas)
		fmt.Fprintln(scoreText, score)
		scoreText.Draw(win, pixel.IM.Scaled(win.Bounds().Center(), camZoom).Moved(playerTruePos))
		animate(&ringicon, dt, 12, 7, 0, ringsheet, ringFrames)
		ringicon.me.Draw(win, pixel.IM.Scaled(win.Bounds().Center(), camZoom/3).Moved(playerTruePos.Add(pixel.V(50, 39))))

		win.Update() //update window

		frames++ //keep track of framerate and display it on window title
		select {
		case <-second:
			win.SetTitle(fmt.Sprintf("%s | FPS: %d", cfg.Title, frames))
			frames = 0
		default:
		}

	}
}

/*
	Checks collisions of anims against anims
*/
func animCollisions(subject *anim) {
	for i := 0; i < len(animsList); i++ {
		//check if distance apart greater than or equal to sum of the two radii
		if distance(subject.col.center, animsList[i].col.center) <= subject.col.radius+animsList[i].col.radius {
			if animsList[i].tag == "ring" {
				animsList = removeAnim(i) //collect ring
				score++
			}
		}
	}
}

/*
	Self explanatory, removes anim from animsList and refactors
*/
func removeAnim(i int) []anim {
	animsList[i] = animsList[len(animsList)-1]
	return animsList[:len(animsList)-1]
}

/*
	Tests collisions of anims against barriers
*/
func checkCollision(subject *anim) int {
	circ := subject.col
	totalCollisions := 0
	for _, line := range barriers {
		var lineMinX float64
		var lineMinY float64
		var lineMaxX float64 //get bounds of line
		var lineMaxY float64
		buffer := 7.0            //makes it smoother around corners
		if line.A.X > line.B.X { //find the max and min bounds of the line
			lineMaxX = line.A.X + circ.radius - buffer
			lineMinX = line.B.X - circ.radius + buffer
		} else {
			lineMaxX = line.B.X + circ.radius - buffer
			lineMinX = line.A.X - circ.radius + buffer
		}
		if line.A.Y > line.B.Y {
			lineMaxY = line.A.Y + circ.radius - buffer
			lineMinY = line.B.Y - circ.radius + buffer
		} else {
			lineMaxY = line.B.Y + circ.radius - buffer
			lineMinY = line.A.Y - circ.radius + buffer
		}

		//slope = y₂-y₁ / x₂-x₁
		num := line.B.Y - line.A.Y  //numerator
		dnom := line.B.X - line.A.X //denominator
		if dnom != 0 && num != 0 {  //don't divide by 0! and make sure slope isn't 0
			if subject.col.center.X >= lineMinX && //if subject is within the bounds of the line
				subject.col.center.X <= lineMaxX &&
				subject.col.center.Y >= lineMinY &&
				subject.col.center.Y <= lineMaxY {
				var slope float64
				//Two-point formula to find the equation of our line
				//y-y₁ = y₂-y₁ / x₂-x₁ * (x-x₂)
				slope = num / dnom
				origX := line.A.X
				origY := (slope * (origX - line.B.X)) + line.B.Y
				//slope intercept form
				intercept := origY - (origX * slope)

				//slope and intercept of our second line, which passes thru center of circle
				recipSlope := 1 / -slope //opposite reciprocal of slope

				recipIntercept := circ.center.Y - (circ.center.X * recipSlope) //intercept of reciprocal line

				//find point that intersects two lines, Cramer's formula
				//line1: a₁X + b₁Y + c₁ = 0
				//line2: a₂X + b₂Y + c₂ = 0
				//(x,y) = ( b₁c₂-b₂c₁/a₁b₂-a₂b₁ , c₁a₂-c₂a₁/a₁b₂-a₂b₁)
				finalX := ((-1 * recipIntercept) - (-1 * intercept)) / ((slope * -1) - (recipSlope * -1))
				finalY := ((intercept * recipSlope) - (recipIntercept * slope)) / ((slope * -1) - (recipSlope * -1))

				intersectionPoint := pixel.V(finalX, finalY)
				//intersection between the collider line and a perpendicular line that passes through the center of the circle

				//find distance between new point and center of circle
				dist := distance(intersectionPoint, circ.center)
				//check collision
				if dist <= circ.radius {
					//collision occurred!
					totalCollisions++
					//find the length of the part of the radius that crossed the line
					crossOver := circ.radius - dist
					//pythagorean theorem: a² + b² = c²
					//we need to find a and b, knowing c (c is our Crossover value)
					var a float64
					var b float64
					//we know the equation of the hypotenuse line has a slope of the value recipSlope
					//and because of how right triangles work, b/a = m slope
					// thus a = mb
					//combine that with the pythagorean theorem and simplify to get the following equation:
					//b = c/√m²+1, solve for b
					b = crossOver / math.Sqrt(math.Pow(recipSlope, 2)+1)
					//use b to solve for a
					a = recipSlope * b
					//now we can add these values to our subject to put the circle edge right on the edge of the line
					if slope > 0 {
						if circ.center.Y > intersectionPoint.Y { //if you're above the line it's swapped. Don't ask me why
							subject.pos.X += b
							subject.pos.Y += a
						} else {
							subject.pos.X += a
							subject.pos.Y += b
						}
					} else if slope < 0 { //if your slope is negative you subtract instead of add
						if circ.center.Y > intersectionPoint.Y {
							subject.pos.X -= a
							subject.pos.Y -= b
						} else {
							subject.pos.X += b //this one was a curveball... throws off the pattern a little
							subject.pos.Y += a
						}
					}

				}
			}
		} else if num == 0 { //line is horizontal
			if subject.col.center.X >= lineMinX &&
				subject.col.center.X <= lineMaxX {
				finalX := subject.col.center.X
				finalY := line.A.Y

				intersectionPoint := pixel.V(finalX, finalY)
				//intersection between the collider line and a perpendicular line that passes through the center of the circle
				//find distance between new point and center of circle
				dist := distance(intersectionPoint, circ.center)
				if dist <= circ.radius {
					//collision!
					totalCollisions++
					crossOver := circ.radius - dist
					if circ.center.Y > intersectionPoint.Y {
						subject.pos.Y -= crossOver
					} else {
						subject.pos.Y += crossOver
					}

				}
			}
		} else { //line is Vertical
			if subject.col.center.Y >= lineMinY &&
				subject.col.center.Y <= lineMaxY {
				finalX := line.B.X
				finalY := subject.col.center.Y

				intersectionPoint := pixel.V(finalX, finalY)
				//intersection between the collider line and a perpendicular line that passes through the center of the circle
				//find distance between new point and center of circle
				dist := distance(intersectionPoint, circ.center)
				if dist <= circ.radius {
					//collision!
					totalCollisions++
					crossOver := circ.radius - dist
					if circ.center.X > intersectionPoint.X {
						subject.pos.X -= crossOver
					} else {
						subject.pos.X += crossOver
					}

				}
			}
		}
	}
	return totalCollisions
}

/*
	Finds the in world distance between two points
*/
func distance(point1 pixel.Vec, point2 pixel.Vec) float64 {
	// distance = √(x₂ - x₁)² + (y₂ - y₁)²,
	dist := math.Sqrt(math.Pow(point1.X-point2.X, 2) + math.Pow(point1.Y-point2.Y, 2))
	return dist
}

/*
	Checks if a float is between 2 other floats
*/
func between(min float64, i float64, max float64) bool {
	if (i >= min) && (i <= max) {
		return true
	} else {
		return false
	}
}

/*
	Moves the goblin where he needs to go. Returns persistent info between calls so the goblin can keep track
	of where he is and what he's doing.
*/
func goblinMovement(goblin *anim, dt float64, playerpos pixel.Vec, goblinfo *goblinKnowledge) {
	buffer := 10.0 //don't need to be exact, just in a range
	//move goblin towards player
	if goblinfo.follow {
		if playerpos.X+buffer < goblin.pos.X || playerpos.X+buffer < goblin.pos.X && goblin.pos.X < playerpos.X-buffer {
			goblin.scale = pixel.V(1, 1)
			if playerpos.Y+buffer < goblin.pos.Y || playerpos.Y+buffer < goblin.pos.Y && goblin.pos.Y < playerpos.Y-buffer {
				goblin.dir = SW
			} else if playerpos.Y-buffer > goblin.pos.Y || playerpos.Y-buffer > goblin.pos.Y && goblin.pos.Y > playerpos.X+buffer {
				goblin.dir = NW
			} else {
				goblin.dir = W
			}
		} else if playerpos.X-buffer > goblin.pos.X || playerpos.X-buffer > goblin.pos.X && goblin.pos.X > playerpos.X+buffer {
			goblin.scale = pixel.V(-1, 1)
			if playerpos.Y+buffer < goblin.pos.Y || playerpos.Y+buffer < goblin.pos.Y && goblin.pos.Y < playerpos.Y-buffer {
				goblin.dir = SE
			} else if playerpos.Y-buffer > goblin.pos.Y || playerpos.Y-buffer > goblin.pos.Y && goblin.pos.Y > playerpos.X+buffer {
				goblin.dir = NE
			} else {
				goblin.dir = E
			}
		} else if playerpos.Y+buffer < goblin.pos.Y || playerpos.Y+buffer < goblin.pos.Y && goblin.pos.Y < playerpos.Y-buffer {
			goblin.dir = S
		} else if playerpos.Y-buffer > goblin.pos.Y || playerpos.Y-buffer > goblin.pos.Y && goblin.pos.Y > playerpos.X+buffer {
			goblin.dir = N
		}
	}

	closeEnoughToFollow := true

	if distance(goblin.pos, playerpos) > 500 { //if too far away, goblins stop following.
		closeEnoughToFollow = false
	}

	if checkCollision(goblin) > 0 { //goblin is colliding
		goblin.dir = nextDir(goblinfo.LastDir) //rotate 90 degrees
		goblinfo.follow = false
		goblinfo.timeSpent = 0 //reset time
	} else {
		goblinfo.timeSpent += dt //increment time since last collision
	}
	if goblinfo.timeSpent > 1.0 && closeEnoughToFollow {
		//if youre not colliding for more than a second and youre close enough, start following again
		goblinfo.follow = true
	}

	dirs := strings.Split(string(goblin.dir), ",")
	xVal, _ := strconv.ParseFloat(dirs[0], 32)
	yVal, _ := strconv.ParseFloat(dirs[1], 32)
	goblinfo.offset, _ = strconv.Atoi(dirs[2]) //get offset for animation row we want to use
	if closeEnoughToFollow {
		goblin.pos.X -= xVal * goblin.speed * dt //calculate movement of character
		goblin.pos.Y -= yVal * goblin.speed * dt
	}

	goblinfo.LastDir = goblin.dir //store this for next time

	goblin.sortLayer = int(goblin.pos.Y - 60)                  //offset so its at the bottom of the character
	goblin.col.center = pixel.V(goblin.pos.X, goblin.pos.Y-60) //put collider where it needs to be
}

/*
	Figures out what frame needs to be displayed for an anim based off of framerate. Keeps track of what animation in
	what spritesheet needs to be used
*/
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

/*
	Reads in a text file and stores lines in our barriers array.
*/
func ReadLayout() {

	file, err := os.Open("layout.txt") //open to read

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close() //ensure file is closed

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineElems := strings.Split(scanner.Text(), ",") //split on commas
		if len(lineElems) == 5 {
			pointAX, _ := strconv.ParseFloat(lineElems[0], 64)
			pointAY, _ := strconv.ParseFloat(lineElems[1], 64)
			pointBX, _ := strconv.ParseFloat(lineElems[2], 64)
			pointBY, _ := strconv.ParseFloat(lineElems[3], 64)
			buffer := 5.0
			//horizontal and vertical barriers perform better than slanted, so if its close enough just make it flat
			if between(pointBX-buffer, pointAX, pointBX+buffer) { //if point is close enough, make it the same.
				pointAX = pointBX
			}
			if between(pointBY-buffer, pointAY, pointBY+buffer) { //if point is close enough, make it the same.
				pointAY = pointBY
			}
			newbar := line{pixel.V(pointAX, pointAY), pixel.V(pointBX, pointBY)}
			barriers = append(barriers, newbar)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

/*
	Reads in item info from the file
*/
func ReadItems() {

	file, err := os.Open("items.txt") //open to read

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineElems := strings.Split(scanner.Text(), ",") //split on commas
		goblinCount := 0                                //count goblins to keep track of whos who when we later assign brains to them
		if len(lineElems) == 4 {
			tag := lineElems[0]
			X, _ := strconv.ParseFloat(lineElems[1], 64)
			Y, _ := strconv.ParseFloat(lineElems[2], 64)
			pos := pixel.V(X, Y)
			if tag == "ring" { //its a ring
				newring := anim{*pixel.NewSprite(ringsheet, ringFrames[0]), tag, 0, 0,
					circle{pixel.ZV, 10}, pos, pixel.V(1, 1),
					S, 0, int(Y)}
				animsList = append(animsList, newring) //add new ring to animslist
			} else if tag == "goblin" { //its a goblin!
				newgob := anim{*pixel.NewSprite(goblinsheet, goblinFrames[0]), string(goblinCount), 0, 0,
					circle{pixel.ZV, 15}, pos, pixel.V(1, 1),
					S, 80, int(Y) - 60}
				animsList = append(animsList, newgob)    //add new goblin to animslist
				brain := goblinKnowledge{S, true, 0, 10} //create a new goblinKnowledge
				goblinfo = append(goblinfo, brain)
				goblinCount++
			} else if tag == "ted" { //its a ring
				newted := anim{*pixel.NewSprite(tedsheet, tedFrames[0]), tag, 0, 0,
					circle{pixel.ZV, 10}, pos, pixel.V(1, 1),
					S, 0, int(Y - 50)}
				animsList = append(animsList, newted) //add to animslist
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

/*
	Loads a basic Go picture as a pixel picture
*/
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

func main() {
	pixelgl.Run(run)
}
