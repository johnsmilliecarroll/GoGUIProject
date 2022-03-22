package main

import (
	"bufio"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
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

type anim struct { //animated sprite
	me        pixel.Sprite //the sprite associated with the struct
	tag       string
	index     int     //current frame the animation is on
	quantum   float64 //amount of time until next frame
	col       circle  //collider
	pos       pixel.Vec
	scale     pixel.Vec
	sortLayer int
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

var barriers []line

var animsList []anim

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

	bgimg, err := loadPicture("sprites/map.png")
	if err != nil {
		panic(err)
	}
	bgimg2, err := loadPicture("sprites/mapoverlay.png")
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
	ringScale := 43.0
	var ringFrames []pixel.Rect
	for y := ringsheet.Bounds().Min.Y; y < ringsheet.Bounds().Max.Y; y += ringScale {
		for x := ringsheet.Bounds().Min.X; x < ringsheet.Bounds().Max.X; x += ringScale {
			ringFrames = append(ringFrames, pixel.R(x, y, x+ringScale, y+ringScale))
		}
	}

	ReadLayout() //load in level barriers

	var (
		player = anim{*pixel.NewSprite(idlesheet, idleFrames[0]), "player", 0, 0,
			circle{pixel.ZV, 20}, pixel.ZV, pixel.V(1, 1), 0}
		playerSpeed      = 150.0
		playerDir        = S
		lastDir          = S
		playerScale      = 1.0
		playerMoving     = false
		playerAnimOffset = 0
		playerSheet      = idlesheet
		playerFrames     = idleFrames
		playerAnimSpeed  = 15
		playerFrameCount = 8

		ring1 = anim{*pixel.NewSprite(ringsheet, ringFrames[0]), "ring", 0, 0,
			circle{pixel.ZV, 10}, pixel.V(500, 300), pixel.V(1, 1), 300}

		ring2 = anim{*pixel.NewSprite(ringsheet, ringFrames[0]), "ring", 0, 0,
			circle{pixel.ZV, 10}, pixel.V(700, 700), pixel.V(1, 1), 700}

		background       = pixel.NewSprite(bgimg, bgimg.Bounds())
		bgOverlay        = pixel.NewSprite(bgimg2, bgimg2.Bounds())
		backgroundOffset = pixel.V(-80, -520)

		camZoom = 2.0
		//camZoomSpeed = 1.2

		DEBUG = false
	)
	animsList = append(animsList, player)
	animsList = append(animsList, ring1)
	animsList = append(animsList, ring2)

	last := time.Now()
	for !win.Closed() {
		dt := time.Since(last).Seconds() //delta time
		last = time.Now()

		cam := pixel.IM.Scaled(win.Bounds().Center().Sub(player.pos), camZoom).Moved(player.pos)
		win.SetMatrix(cam)

		//camZoom *= math.Pow(camZoomSpeed, win.MouseScroll().Y)

		//region PLAYER MOVEMENT
		if win.Pressed(pixelgl.KeyLeft) || win.Pressed(pixelgl.KeyA) { //test against key presses and all possible key combinations
			playerMoving = true
			playerScale = 1
			if win.Pressed(pixelgl.KeyDown) || win.Pressed(pixelgl.KeyS) {
				playerDir = SW
			} else if win.Pressed(pixelgl.KeyUp) || win.Pressed(pixelgl.KeyW) {
				playerDir = NW
			} else {
				playerDir = W
			}
		} else if win.Pressed(pixelgl.KeyRight) || win.Pressed(pixelgl.KeyD) {
			playerMoving = true
			playerScale = -1
			if win.Pressed(pixelgl.KeyDown) || win.Pressed(pixelgl.KeyS) {
				playerDir = SE
			} else if win.Pressed(pixelgl.KeyUp) || win.Pressed(pixelgl.KeyW) {
				playerDir = NE
			} else {
				playerDir = E
			}
		} else if win.Pressed(pixelgl.KeyDown) || win.Pressed(pixelgl.KeyS) {
			playerMoving = true
			playerDir = S
		} else if win.Pressed(pixelgl.KeyUp) || win.Pressed(pixelgl.KeyW) {
			playerMoving = true
			playerDir = N
		}
		if playerMoving { //convert direction from string to movement
			dirs := strings.Split(string(playerDir), ",")
			xVal, _ := strconv.ParseFloat(dirs[0], 32)
			yVal, _ := strconv.ParseFloat(dirs[1], 32)
			OffsetVal, _ := strconv.Atoi(dirs[2])   //get offset for animation row we want to use
			player.pos.X += xVal * playerSpeed * dt //calculate movement of character
			player.pos.Y += yVal * playerSpeed * dt
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

		// if keys are released, player isn't moving.
		if win.JustReleased(pixelgl.KeyLeft) || win.JustReleased(pixelgl.KeyA) ||
			win.JustReleased(pixelgl.KeyRight) || win.JustReleased(pixelgl.KeyD) ||
			win.JustReleased(pixelgl.KeyDown) || win.JustReleased(pixelgl.KeyS) ||
			win.JustReleased(pixelgl.KeyUp) || win.JustReleased(pixelgl.KeyW) {
			playerMoving = false
		}
		//endregion

		win.Clear(colornames.Coral) //refresh window, set color
		background.Draw(win, pixel.IM.Scaled(pixel.ZV, 1).Moved(win.Bounds().Center().Sub(backgroundOffset)))

		//figure out the current frame of the character and draw it
		animate(&player, dt, playerAnimSpeed, playerFrameCount, playerAnimOffset, playerSheet, playerFrames)
		playerTruePos := win.Bounds().Center().Sub(player.pos)
		//player.me.Draw(win, pixel.IM.ScaledXY(pixel.ZV, player.scale).Moved(win.Bounds().Center().Sub(player.pos)))
		player.scale = pixel.V(playerScale, 1)
		player.sortLayer = int(playerTruePos.Y - 35) //offset so its at the bottom of the character
		player.col.center = pixel.V(playerTruePos.X, playerTruePos.Y-20)
		circleCollision(&player)
		animCollisions(&player)

		//sort sprites to be drawn according to sorting layer.
		sort.Slice(animsList, func(j, i int) bool {
			return animsList[i].sortLayer < animsList[j].sortLayer
		})

		for i := 0; i < len(animsList); i++ {
			if animsList[i].tag == "player" {
				animsList[i].sortLayer = player.sortLayer
				player.me.Draw(win, pixel.IM.ScaledXY(pixel.ZV, player.scale).Moved(win.Bounds().Center().Sub(player.pos)))
			} else {
				animate(&animsList[i], dt, 12, 7, 0, ringsheet, ringFrames)
				animsList[i].me.Draw(win, pixel.IM.Scaled(pixel.ZV, 1).Moved(animsList[i].pos))
				animsList[i].col.center = animsList[i].pos
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
				imd.Push(line.A)
				imd.Push(line.B)
				imd.Line(2)
			}

			imd.Draw(win)
		}

		win.Update() //update window

	}
}

func animCollisions(subject *anim) {
	for i := 0; i < len(animsList); i++ {
		if Distance(subject.col.center, animsList[i].col.center) <= subject.col.radius+animsList[i].col.radius {
			if animsList[i].tag == "ring" {
				animsList = removeAnim(i)
			}
		}
	}
}

func removeAnim(i int) []anim {
	animsList[i] = animsList[len(animsList)-1]
	return animsList[:len(animsList)-1]
}

func circleCollision(subject *anim) {
	circ := subject.col
	for _, line := range barriers {
		var lineMinX float64
		var lineMinY float64
		var lineMaxX float64 //get bounds of line
		var lineMaxY float64
		if line.A.X > line.B.X { //find the max and min bounds of the line
			lineMaxX = line.A.X + circ.radius
			lineMinX = line.B.X - circ.radius
		} else {
			lineMaxX = line.B.X + circ.radius
			lineMinX = line.A.X - circ.radius
		}
		if line.A.Y > line.B.Y {
			lineMaxY = line.A.Y + circ.radius
			lineMinY = line.B.Y - circ.radius
		} else {
			lineMaxY = line.B.Y + circ.radius
			lineMinY = line.A.Y - circ.radius
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
				dist := Distance(intersectionPoint, circ.center)
				//check collision
				if dist <= circ.radius {
					//collision occurred!
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
				dist := Distance(intersectionPoint, circ.center)
				if dist <= circ.radius {
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
				dist := Distance(intersectionPoint, circ.center)
				if dist <= circ.radius {
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
}

func Distance(point1 pixel.Vec, point2 pixel.Vec) float64 {
	// distance = √(x₂ - x₁)² + (y₂ - y₁)²,
	dist := math.Sqrt(math.Pow(point1.X-point2.X, 2) + math.Pow(point1.Y-point2.Y, 2))
	return dist
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

func ReadLayout() {

	file, err := os.Open("layout.txt") //open to read

	if err != nil {
		log.Fatal(err)
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lineElems := strings.Split(scanner.Text(), ",") //split on commas
		if len(lineElems) == 5 {
			pointAX, _ := strconv.ParseFloat(lineElems[0], 64)
			pointAY, _ := strconv.ParseFloat(lineElems[1], 64)
			pointBX, _ := strconv.ParseFloat(lineElems[2], 64)
			pointBY, _ := strconv.ParseFloat(lineElems[3], 64)
			newbar := line{pixel.V(pointAX, pointAY), pixel.V(pointBX, pointBY)}
			barriers = append(barriers, newbar)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
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

func main() {
	pixelgl.Run(run)
}
