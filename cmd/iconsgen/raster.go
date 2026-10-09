package main

import (
	"image"
	"image/color"
	"math"
)

type point struct{ x, y float64 }

// canvas accumulates antialiased stroke coverage. Strokes are unioned with max, so crossing
// strokes never get darker than one stroke, which keeps the icon strictly one-colour.
type canvas struct {
	size        int
	strokeWidth float64
	coverage    []float64
}

func newCanvas(size int, strokeWidth float64) *canvas {
	return &canvas{size: size, strokeWidth: strokeWidth, coverage: make([]float64, size*size)}
}

// strokePath draws a polyline with round caps and joins of the given line width.
func (c *canvas) strokePath(points []point, width float64) {
	if len(points) == 1 {
		points = append(points, points[0])
	}
	for i := 1; i < len(points); i++ {
		c.strokeSegment(points[i-1], points[i], width)
	}
}

// strokeSegment gives each pixel the coverage 0.5 + (halfWidth - distance), clamped to 0..1:
// a one pixel wide linear edge ramp, which is what a box-filtered antialiased edge looks like.
func (c *canvas) strokeSegment(from, to point, width float64) {
	half := width / 2
	reach := half + 1
	minX, maxX := c.clampIndex(math.Min(from.x, to.x)-reach), c.clampIndex(math.Max(from.x, to.x)+reach)
	minY, maxY := c.clampIndex(math.Min(from.y, to.y)-reach), c.clampIndex(math.Max(from.y, to.y)+reach)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			distance := distanceToSegment(point{float64(x) + 0.5, float64(y) + 0.5}, from, to)
			covered := math.Max(0, math.Min(1, 0.5+half-distance))
			if index := y*c.size + x; covered > c.coverage[index] {
				c.coverage[index] = covered
			}
		}
	}
}

func (c *canvas) clampIndex(value float64) int {
	return int(math.Max(0, math.Min(float64(c.size-1), math.Floor(value))))
}

func distanceToSegment(p, from, to point) float64 {
	dx, dy := to.x-from.x, to.y-from.y
	lengthSquared := dx*dx + dy*dy
	t := 0.0
	if lengthSquared > 0 {
		t = math.Max(0, math.Min(1, ((p.x-from.x)*dx+(p.y-from.y)*dy)/lengthSquared))
	}
	return math.Hypot(p.x-(from.x+t*dx), p.y-(from.y+t*dy))
}

// image returns the strokes as black pixels whose alpha is the coverage.
func (c *canvas) image() *image.NRGBA {
	result := image.NewNRGBA(image.Rect(0, 0, c.size, c.size))
	for y := 0; y < c.size; y++ {
		for x := 0; x < c.size; x++ {
			alpha := uint8(math.Round(c.coverage[y*c.size+x] * 255))
			result.SetNRGBA(x, y, color.NRGBA{A: alpha})
		}
	}
	return result
}

const arcStepDegrees = 3.0

// arcPoints samples a circle arc; angles are in degrees, increasing clockwise on screen.
func arcPoints(center point, radius, fromDegrees, toDegrees float64) []point {
	steps := int(math.Ceil(math.Abs(toDegrees-fromDegrees) / arcStepDegrees))
	points := make([]point, 0, steps+1)
	for i := 0; i <= steps; i++ {
		angle := (fromDegrees + (toDegrees-fromDegrees)*float64(i)/float64(steps)) * math.Pi / 180
		points = append(points, point{center.x + radius*math.Cos(angle), center.y + radius*math.Sin(angle)})
	}
	return points
}

func circlePoints(center point, radius float64) []point {
	return arcPoints(center, radius, 0, 360)
}

// quadraticPoints samples a quadratic Bezier curve.
func quadraticPoints(from, control, to point) []point {
	const steps = 24
	points := make([]point, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / steps
		u := 1 - t
		points = append(points, point{
			u*u*from.x + 2*u*t*control.x + t*t*to.x,
			u*u*from.y + 2*u*t*control.y + t*t*to.y,
		})
	}
	return points
}

// roundedRectPoints is a closed outline; the first point is repeated at the end.
func roundedRectPoints(left, top, right, bottom, radius float64) []point {
	var points []point
	points = append(points, arcPoints(point{right - radius, top + radius}, radius, -90, 0)...)
	points = append(points, arcPoints(point{right - radius, bottom - radius}, radius, 0, 90)...)
	points = append(points, arcPoints(point{left + radius, bottom - radius}, radius, 90, 180)...)
	points = append(points, arcPoints(point{left + radius, top + radius}, radius, 180, 270)...)
	return append(points, points[0])
}
