// Package svg provides SVG path parsing and rendering for TermGL.
package svg

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// PathCommand represents a single SVG path command.
type PathCommand struct {
	Type     rune      // Command type (M, L, H, V, C, S, Q, T, A, Z)
	Absolute bool      // true for uppercase (absolute), false for lowercase (relative)
	Args     []float64 // Command arguments
}

// Path represents a parsed SVG path.
type Path struct {
	Commands []PathCommand
}

// Point represents a 2D point.
type Point struct {
	X, Y float64
}

// ParsePath parses an SVG path d attribute string.
func ParsePath(d string) (*Path, error) {
	tokens := tokenize(d)
	commands, err := parseTokens(tokens)
	if err != nil {
		return nil, err
	}
	return &Path{Commands: commands}, nil
}

// tokenize splits the path string into tokens.
func tokenize(d string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range d {
		if unicode.IsLetter(r) {
			// Save any pending number
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(r))
		} else if unicode.IsDigit(r) || r == '.' || r == '-' || r == '+' {
			// Handle negative sign after another number (implicit separator)
			if r == '-' && current.Len() > 0 {
				// Check if this is a negative number start or part of exponent
				lastChar := rune(current.String()[current.Len()-1])
				if lastChar != 'e' && lastChar != 'E' {
					tokens = append(tokens, current.String())
					current.Reset()
				}
			}
			current.WriteRune(r)
		} else if r == ',' || unicode.IsSpace(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else if r == 'e' || r == 'E' {
			// Scientific notation
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// parseTokens converts tokens into path commands.
func parseTokens(tokens []string) ([]PathCommand, error) {
	var commands []PathCommand
	var currentCmd rune
	var isAbsolute bool
	var args []float64

	flushCommand := func() {
		if currentCmd == 0 {
			return
		}

		// Create command(s) from accumulated args
		argCounts := map[rune]int{
			'M': 2, 'L': 2, 'H': 1, 'V': 1,
			'C': 6, 'S': 4, 'Q': 4, 'T': 2,
			'A': 7, 'Z': 0,
		}

		cmdUpper := unicode.ToUpper(currentCmd)
		argCount := argCounts[cmdUpper]

		if argCount == 0 {
			commands = append(commands, PathCommand{
				Type:     cmdUpper,
				Absolute: isAbsolute,
				Args:     nil,
			})
		} else {
			// Split args into multiple commands if needed
			for i := 0; i+argCount <= len(args); i += argCount {
				cmdType := cmdUpper
				// M becomes L after first point, m becomes l
				if cmdUpper == 'M' && i > 0 {
					cmdType = 'L'
				}
				commands = append(commands, PathCommand{
					Type:     cmdType,
					Absolute: isAbsolute,
					Args:     args[i : i+argCount],
				})
			}
		}
		args = nil
	}

	for _, token := range tokens {
		if len(token) == 1 && unicode.IsLetter(rune(token[0])) {
			// New command
			flushCommand()
			currentCmd = rune(token[0])
			isAbsolute = unicode.IsUpper(currentCmd)
		} else {
			// Number argument
			val, err := strconv.ParseFloat(token, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", token)
			}
			args = append(args, val)
		}
	}

	flushCommand()
	return commands, nil
}

// ToPoints converts the path to a series of points using linear approximation.
// resolution controls how many points per curve segment.
func (p *Path) ToPoints(resolution int) []Point {
	if resolution < 1 {
		resolution = 10
	}

	var points []Point
	var currentX, currentY float64
	var startX, startY float64 // For closepath
	var lastControlX, lastControlY float64
	var lastCmd rune

	for _, cmd := range p.Commands {
		switch cmd.Type {
		case 'M': // MoveTo
			if cmd.Absolute {
				currentX, currentY = cmd.Args[0], cmd.Args[1]
			} else {
				currentX += cmd.Args[0]
				currentY += cmd.Args[1]
			}
			startX, startY = currentX, currentY
			points = append(points, Point{currentX, currentY})

		case 'L': // LineTo
			var x, y float64
			if cmd.Absolute {
				x, y = cmd.Args[0], cmd.Args[1]
			} else {
				x, y = currentX+cmd.Args[0], currentY+cmd.Args[1]
			}
			points = append(points, Point{x, y})
			currentX, currentY = x, y

		case 'H': // Horizontal line
			var x float64
			if cmd.Absolute {
				x = cmd.Args[0]
			} else {
				x = currentX + cmd.Args[0]
			}
			points = append(points, Point{x, currentY})
			currentX = x

		case 'V': // Vertical line
			var y float64
			if cmd.Absolute {
				y = cmd.Args[0]
			} else {
				y = currentY + cmd.Args[0]
			}
			points = append(points, Point{currentX, y})
			currentY = y

		case 'C': // Cubic Bezier
			var x1, y1, x2, y2, x, y float64
			if cmd.Absolute {
				x1, y1 = cmd.Args[0], cmd.Args[1]
				x2, y2 = cmd.Args[2], cmd.Args[3]
				x, y = cmd.Args[4], cmd.Args[5]
			} else {
				x1, y1 = currentX+cmd.Args[0], currentY+cmd.Args[1]
				x2, y2 = currentX+cmd.Args[2], currentY+cmd.Args[3]
				x, y = currentX+cmd.Args[4], currentY+cmd.Args[5]
			}
			pts := cubicBezier(currentX, currentY, x1, y1, x2, y2, x, y, resolution)
			points = append(points, pts...)
			lastControlX, lastControlY = x2, y2
			currentX, currentY = x, y

		case 'S': // Smooth cubic Bezier
			var x1, y1, x2, y2, x, y float64
			// Reflect previous control point
			if lastCmd == 'C' || lastCmd == 'S' {
				x1 = 2*currentX - lastControlX
				y1 = 2*currentY - lastControlY
			} else {
				x1, y1 = currentX, currentY
			}
			if cmd.Absolute {
				x2, y2 = cmd.Args[0], cmd.Args[1]
				x, y = cmd.Args[2], cmd.Args[3]
			} else {
				x2, y2 = currentX+cmd.Args[0], currentY+cmd.Args[1]
				x, y = currentX+cmd.Args[2], currentY+cmd.Args[3]
			}
			pts := cubicBezier(currentX, currentY, x1, y1, x2, y2, x, y, resolution)
			points = append(points, pts...)
			lastControlX, lastControlY = x2, y2
			currentX, currentY = x, y

		case 'Q': // Quadratic Bezier
			var x1, y1, x, y float64
			if cmd.Absolute {
				x1, y1 = cmd.Args[0], cmd.Args[1]
				x, y = cmd.Args[2], cmd.Args[3]
			} else {
				x1, y1 = currentX+cmd.Args[0], currentY+cmd.Args[1]
				x, y = currentX+cmd.Args[2], currentY+cmd.Args[3]
			}
			pts := quadraticBezier(currentX, currentY, x1, y1, x, y, resolution)
			points = append(points, pts...)
			lastControlX, lastControlY = x1, y1
			currentX, currentY = x, y

		case 'T': // Smooth quadratic Bezier
			var x1, y1, x, y float64
			// Reflect previous control point
			if lastCmd == 'Q' || lastCmd == 'T' {
				x1 = 2*currentX - lastControlX
				y1 = 2*currentY - lastControlY
			} else {
				x1, y1 = currentX, currentY
			}
			if cmd.Absolute {
				x, y = cmd.Args[0], cmd.Args[1]
			} else {
				x, y = currentX+cmd.Args[0], currentY+cmd.Args[1]
			}
			pts := quadraticBezier(currentX, currentY, x1, y1, x, y, resolution)
			points = append(points, pts...)
			lastControlX, lastControlY = x1, y1
			currentX, currentY = x, y

		case 'A': // Arc
			var rx, ry, xAxisRotation float64
			var largeArcFlag, sweepFlag bool
			var x, y float64

			rx, ry = cmd.Args[0], cmd.Args[1]
			xAxisRotation = cmd.Args[2]
			largeArcFlag = cmd.Args[3] != 0
			sweepFlag = cmd.Args[4] != 0

			if cmd.Absolute {
				x, y = cmd.Args[5], cmd.Args[6]
			} else {
				x, y = currentX+cmd.Args[5], currentY+cmd.Args[6]
			}

			pts := arcToBezier(currentX, currentY, rx, ry, xAxisRotation, largeArcFlag, sweepFlag, x, y, resolution)
			points = append(points, pts...)
			currentX, currentY = x, y

		case 'Z': // Close path
			if currentX != startX || currentY != startY {
				points = append(points, Point{startX, startY})
			}
			currentX, currentY = startX, startY
		}

		lastCmd = cmd.Type
	}

	return points
}

// BoundingBox returns the bounding box of the path.
func (p *Path) BoundingBox() (minX, minY, maxX, maxY float64) {
	points := p.ToPoints(20)
	if len(points) == 0 {
		return 0, 0, 0, 0
	}

	minX, minY = points[0].X, points[0].Y
	maxX, maxY = points[0].X, points[0].Y

	for _, pt := range points[1:] {
		if pt.X < minX {
			minX = pt.X
		}
		if pt.X > maxX {
			maxX = pt.X
		}
		if pt.Y < minY {
			minY = pt.Y
		}
		if pt.Y > maxY {
			maxY = pt.Y
		}
	}

	return
}

// Length calculates the approximate length of the path.
func (p *Path) Length() float64 {
	points := p.ToPoints(50)
	if len(points) < 2 {
		return 0
	}

	var length float64
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		length += math.Sqrt(dx*dx + dy*dy)
	}

	return length
}

// PointAt returns the point at a given t value (0-1) along the path.
func (p *Path) PointAt(t float64) Point {
	points := p.ToPoints(100)
	if len(points) == 0 {
		return Point{}
	}
	if len(points) == 1 {
		return points[0]
	}

	// Calculate total length and target distance
	totalLength := p.Length()
	targetDist := t * totalLength

	// Walk along path to find point
	var dist float64
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		segLen := math.Sqrt(dx*dx + dy*dy)

		if dist+segLen >= targetDist {
			// Interpolate within this segment
			remaining := targetDist - dist
			ratio := remaining / segLen
			return Point{
				X: points[i-1].X + ratio*dx,
				Y: points[i-1].Y + ratio*dy,
			}
		}
		dist += segLen
	}

	return points[len(points)-1]
}

// TangentAt returns the tangent direction at a given t value (0-1).
func (p *Path) TangentAt(t float64) Point {
	points := p.ToPoints(100)
	if len(points) < 2 {
		return Point{1, 0}
	}

	totalLength := p.Length()
	targetDist := t * totalLength

	var dist float64
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		segLen := math.Sqrt(dx*dx + dy*dy)

		if dist+segLen >= targetDist || i == len(points)-1 {
			// Normalize tangent
			if segLen > 0 {
				return Point{dx / segLen, dy / segLen}
			}
			return Point{1, 0}
		}
		dist += segLen
	}

	return Point{1, 0}
}

// SubPath returns a subset of the path from t1 to t2 (0-1).
func (p *Path) SubPath(t1, t2 float64) []Point {
	if t1 > t2 {
		t1, t2 = t2, t1
	}

	points := p.ToPoints(100)
	if len(points) < 2 {
		return points
	}

	totalLength := p.Length()
	startDist := t1 * totalLength
	endDist := t2 * totalLength

	var result []Point
	var dist float64

	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		segLen := math.Sqrt(dx*dx + dy*dy)

		segStart := dist
		segEnd := dist + segLen

		if segEnd >= startDist && segStart <= endDist {
			// This segment overlaps with our range
			clampStart := math.Max(startDist, segStart)
			clampEnd := math.Min(endDist, segEnd)

			// Add start point
			if len(result) == 0 {
				ratio := (clampStart - segStart) / segLen
				result = append(result, Point{
					X: points[i-1].X + ratio*dx,
					Y: points[i-1].Y + ratio*dy,
				})
			}

			// Add end point
			ratio := (clampEnd - segStart) / segLen
			result = append(result, Point{
				X: points[i-1].X + ratio*dx,
				Y: points[i-1].Y + ratio*dy,
			})
		}

		dist += segLen
	}

	return result
}

// Bezier curve helper functions

func cubicBezier(x0, y0, x1, y1, x2, y2, x3, y3 float64, steps int) []Point {
	points := make([]Point, 0, steps)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		u := 1 - t
		tt := t * t
		uu := u * u
		uuu := uu * u
		ttt := tt * t

		x := uuu*x0 + 3*uu*t*x1 + 3*u*tt*x2 + ttt*x3
		y := uuu*y0 + 3*uu*t*y1 + 3*u*tt*y2 + ttt*y3
		points = append(points, Point{x, y})
	}
	return points
}

func quadraticBezier(x0, y0, x1, y1, x2, y2 float64, steps int) []Point {
	points := make([]Point, 0, steps)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		u := 1 - t

		x := u*u*x0 + 2*u*t*x1 + t*t*x2
		y := u*u*y0 + 2*u*t*y1 + t*t*y2
		points = append(points, Point{x, y})
	}
	return points
}

func arcToBezier(x0, y0, rx, ry, xAxisRotation float64, largeArc, sweep bool, x, y float64, steps int) []Point {
	// Simplified arc approximation using line segments
	// A full implementation would convert to center parameterization and then to bezier curves

	points := make([]Point, 0, steps)

	// Handle degenerate cases
	if rx == 0 || ry == 0 {
		return []Point{{x, y}}
	}

	// Calculate center and angles (simplified)
	dx := (x0 - x) / 2
	dy := (y0 - y) / 2

	// Rotate
	cos := math.Cos(xAxisRotation * math.Pi / 180)
	sin := math.Sin(xAxisRotation * math.Pi / 180)
	x1p := cos*dx + sin*dy
	y1p := -sin*dx + cos*dy

	// Calculate center
	rx2 := rx * rx
	ry2 := ry * ry
	x1p2 := x1p * x1p
	y1p2 := y1p * y1p

	// Ensure radii are large enough
	lambda := x1p2/rx2 + y1p2/ry2
	if lambda > 1 {
		scale := math.Sqrt(lambda)
		rx *= scale
		ry *= scale
		rx2 = rx * rx
		ry2 = ry * ry
	}

	sq := math.Sqrt(math.Max(0, (rx2*ry2-rx2*y1p2-ry2*x1p2)/(rx2*y1p2+ry2*x1p2)))
	if largeArc == sweep {
		sq = -sq
	}

	cxp := sq * rx * y1p / ry
	cyp := -sq * ry * x1p / rx

	cx := cos*cxp - sin*cyp + (x0+x)/2
	cy := sin*cxp + cos*cyp + (y0+y)/2

	// Calculate angles
	startAngle := math.Atan2((y1p-cyp)/ry, (x1p-cxp)/rx)
	endAngle := math.Atan2((-y1p-cyp)/ry, (-x1p-cxp)/rx)

	deltaAngle := endAngle - startAngle
	if sweep && deltaAngle < 0 {
		deltaAngle += 2 * math.Pi
	} else if !sweep && deltaAngle > 0 {
		deltaAngle -= 2 * math.Pi
	}

	// Generate points along arc
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		angle := startAngle + t*deltaAngle

		px := rx * math.Cos(angle)
		py := ry * math.Sin(angle)

		// Rotate back
		finalX := cos*px - sin*py + cx
		finalY := sin*px + cos*py + cy

		points = append(points, Point{finalX, finalY})
	}

	return points
}

// String returns the path as an SVG d attribute string.
func (p *Path) String() string {
	var sb strings.Builder

	for _, cmd := range p.Commands {
		if cmd.Absolute {
			sb.WriteRune(cmd.Type)
		} else {
			sb.WriteRune(unicode.ToLower(cmd.Type))
		}

		for i, arg := range cmd.Args {
			if i > 0 {
				sb.WriteRune(' ')
			}
			sb.WriteString(fmt.Sprintf("%g", arg))
		}
		sb.WriteRune(' ')
	}

	return strings.TrimSpace(sb.String())
}
