package main

import "math"

const (
	iconSize    = 96
	strokeWidth = 5.5
	center      = iconSize / 2.0
	iconMargin  = 8 // no ink closer to the edge than this

	// Nickel's More-tab row icons (beta_features.png, help, notebooks… in the nickel binary's
	// resources, 4.38.23828): 50x50, black on transparent, ~2 px strokes, ink from 2 to 47.
	moreRowIconSize   = 50
	moreRowStroke     = 2.25
	moreRowIconMargin = 2
)

// style is the pixel size, stroke weight and minimum margin of one icon family.
type style struct {
	size   int
	stroke float64
	margin int
}

var (
	drawerStyle  = style{size: iconSize, stroke: strokeWidth, margin: iconMargin}
	moreRowStyle = style{size: moreRowIconSize, stroke: moreRowStroke, margin: moreRowIconMargin}
)

// icon is one generated file: its path below the repo root, its style and its drawing.
type icon struct {
	path  string
	style style
	draw  func(*canvas)
}

var icons = []icon{
	{"apps/localsend/icon.png", drawerStyle, drawReceive},
	{"apps/wallpaper/icon.png", drawerStyle, drawPicture},
	{"apps/library/icon.png", drawerStyle, drawBooks},
	{"apps/ghsync/icon.png", drawerStyle, drawBranch},
	{"host/icons/refresh.png", drawerStyle, drawRefresh},
	{"host/icons/updates.png", drawerStyle, drawUpdate},
	// The "Apps" row in Nickel's More tab (mod/drawer/entry_row.cc), next to Nickel's own rows.
	{"host/icons/apps.png", moreRowStyle, drawAppsGrid},
}

func render(entry icon) *canvas {
	surface := newCanvas(entry.style.size, entry.style.stroke)
	entry.draw(surface)
	return surface
}

func (c *canvas) stroke(points ...point) { c.strokePath(points, c.strokeWidth) }

// arrowDown is a shaft with a chevron head whose tip is at (x, tipY).
func (c *canvas) arrowDown(x, topY, tipY, headSize float64) {
	c.stroke(point{x, topY}, point{x, tipY})
	c.stroke(point{x - headSize, tipY - headSize}, point{x, tipY}, point{x + headSize, tipY - headSize})
}

// drawReceive: arrow dropping into an open tray.
func drawReceive(c *canvas) {
	c.arrowDown(center, 14, 58, 16)
	c.stroke(point{14, 58}, point{14, 80}, point{82, 80}, point{82, 58})
}

// drawPicture: framed picture with a mountain ridge and a sun.
func drawPicture(c *canvas) {
	c.stroke(roundedRectPoints(13, 17, 83, 79, 9)...)
	c.stroke(point{22, 68}, point{39, 47}, point{52, 62}, point{60, 54}, point{74, 68})
	c.stroke(circlePoints(point{65, 36}, 5.5)...)
}

// drawBooks: two upright books and one leaning on them.
func drawBooks(c *canvas) {
	c.stroke(rectPoints(12, 18, 26, 80)...)
	c.stroke(point{12, 32}, point{26, 32})
	c.stroke(rectPoints(32, 18, 46, 80)...)
	c.stroke(point{32, 32}, point{46, 32})
	c.stroke(leaningBook(point{62, 80}, 15, 54, 10)...)
}

func rectPoints(left, top, right, bottom float64) []point {
	return []point{{left, top}, {right, top}, {right, bottom}, {left, bottom}, {left, top}}
}

// leaningBook is a closed rectangle tilted about its bottom-left corner, top moving left.
func leaningBook(bottomLeft point, width, length, tiltDegrees float64) []point {
	tilt := tiltDegrees * math.Pi / 180
	across := point{width * math.Cos(tilt), -width * math.Sin(tilt)}
	up := point{-length * math.Sin(tilt), -length * math.Cos(tilt)}
	bottomRight := point{bottomLeft.x + across.x, bottomLeft.y + across.y}
	topLeft := point{bottomLeft.x + up.x, bottomLeft.y + up.y}
	topRight := point{bottomRight.x + up.x, bottomRight.y + up.y}
	return []point{bottomLeft, bottomRight, topRight, topLeft, bottomLeft}
}

// drawBranch: git branch, three nodes joined by a trunk and a curved branch.
func drawBranch(c *canvas) {
	const nodeRadius = 8.0
	top, bottom, branch := point{26, 21}, point{26, 75}, point{70, 21}
	c.stroke(circlePoints(top, nodeRadius)...)
	c.stroke(circlePoints(bottom, nodeRadius)...)
	c.stroke(circlePoints(branch, nodeRadius)...)
	c.stroke(point{26, top.y + nodeRadius}, point{26, bottom.y - nodeRadius})
	c.stroke(quadraticPoints(point{70, branch.y + nodeRadius}, point{70, 54}, point{26, 58})...)
}

// drawRefresh: a nearly closed circular arrow, clockwise, head at the end of the arc.
func drawRefresh(c *canvas) {
	const radius, endDegrees = 31.0, 235.0
	c.stroke(arcPoints(point{center, center}, radius, -55, endDegrees)...)
	endRadians := endDegrees * math.Pi / 180
	end := point{center + radius*math.Cos(endRadians), center + radius*math.Sin(endRadians)}
	heading := (endDegrees + 90) * math.Pi / 180 // clockwise tangent
	const headLength, spread = 17.0, 38 * math.Pi / 180
	tip := point{end.x + 4*math.Cos(heading), end.y + 4*math.Sin(heading)}
	for _, side := range []float64{-1, 1} {
		back := heading + math.Pi + side*spread
		c.stroke(tip, point{tip.x + headLength*math.Cos(back), tip.y + headLength*math.Sin(back)})
	}
}

// drawUpdate: down arrow inside a circle.
func drawUpdate(c *canvas) {
	c.stroke(circlePoints(point{center, center}, 34)...)
	c.arrowDown(center, 25, 65, 14)
}

// drawAppsGrid: four rounded squares in a 2x2 grid (the app drawer), at the More-row size.
func drawAppsGrid(c *canvas) {
	const first, side, gap, radius = 4.5, 18.0, 5.0, 3.5
	for _, top := range []float64{first, first + side + gap} {
		for _, left := range []float64{first, first + side + gap} {
			c.stroke(roundedRectPoints(left, top, left+side, top+side, radius)...)
		}
	}
}
