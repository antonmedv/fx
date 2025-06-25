package maze

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand/v2"
	"strings"
)

// Maze cell configurations
// The paths of the maze is represented in the binary representation.
const (
	Up = 1 << iota
	Down
	Left
	Right
)

// The solution path is represented by (Up|Down|Left|Right) << SolutionOffset.
// The user's path is represented by (Up|Down|Left|Right) << VisitedOffset.
const (
	SolutionOffset = 4
	VisitedOffset  = 8
)

// Directions is the set of all the directions
var Directions = []int{Up, Down, Left, Right}

// The differences in the x-y coordinate
var (
	dx = map[int]int{Up: -1, Down: 1, Left: 0, Right: 0}
	dy = map[int]int{Up: 0, Down: 0, Left: -1, Right: 1}
)

// Opposite directions
var Opposite = map[int]int{Up: Down, Down: Up, Left: Right, Right: Left}

// Point on the maze
type Point struct {
	X, Y int
}

// Equal judges the equality of the two points
func (point *Point) Equal(target *Point) bool {
	return point.X == target.X && point.Y == target.Y
}

// Advance the point forward by the argument direction
func (point *Point) Advance(direction int) *Point {
	return &Point{point.X + dx[direction], point.Y + dy[direction]}
}

// Maze represents the configuration of a maze
type Maze struct {
	Directions [][]int
	Height     int
	Width      int
	Start      *Point
	Goal       *Point
	Cursor     *Point
	Solved     bool
	Started    bool
	Finished   bool
	randInt    func() int
}

// NewMaze creates a new maze
func NewMaze(height int, width int, options ...Option) *Maze {
	var directions [][]int
	for x := 0; x < height; x++ {
		directions = append(directions, make([]int, width))
	}
	maze := &Maze{
		Directions: directions,
		Height:     height,
		Width:      width,
		Start:      &Point{0, 0},
		Goal:       &Point{height - 1, width - 1},
		Cursor:     &Point{0, 0},
		randInt:    rand.Int,
	}
	for _, option := range options {
		option(maze)
	}
	return maze
}

// Option is the option for the maze
type Option func(*Maze)

// WithStart sets the start point of the maze
func WithStart(start *Point) Option {
	return func(maze *Maze) {
		maze.Start = start
		maze.Cursor = start
	}
}

// WithGoal sets the goal point of the maze
func WithGoal(goal *Point) Option {
	return func(maze *Maze) {
		maze.Goal = goal
	}
}

// WithRandSource sets the random source of the maze
func WithRandSource(source rand.Source) Option {
	return func(maze *Maze) {
		maze.randInt = rand.New(source).Int
	}
}

// Contains judges whether the argument point is inside the maze or not
func (maze *Maze) Contains(point *Point) bool {
	return 0 <= point.X && point.X < maze.Height && 0 <= point.Y && point.Y < maze.Width
}

// Neighbors gathers the nearest undecided points
func (maze *Maze) Neighbors(point *Point) (neighbors []int) {
	for _, direction := range Directions {
		next := point.Advance(direction)
		if maze.Contains(next) && maze.Directions[next.X][next.Y] == 0 {
			neighbors = append(neighbors, direction)
		}
	}
	return neighbors
}

// Connected judges whether the two points is connected by a path on the maze
func (maze *Maze) Connected(point *Point, target *Point) bool {
	dir := maze.Directions[point.X][point.Y]
	for _, direction := range Directions {
		if dir&direction != 0 {
			next := point.Advance(direction)
			if next.X == target.X && next.Y == target.Y {
				return true
			}
		}
	}
	return false
}

// Next advances the maze path randomly and returns the new point
func (maze *Maze) Next(point *Point) *Point {
	neighbors := maze.Neighbors(point)
	if len(neighbors) == 0 {
		return nil
	}
	direction := neighbors[maze.randInt()%len(neighbors)]
	maze.Directions[point.X][point.Y] |= direction
	next := point.Advance(direction)
	maze.Directions[next.X][next.Y] |= Opposite[direction]
	return next
}

// Generate the maze
func (maze *Maze) Generate() {
	point := maze.Start
	stack := []*Point{point}
	for len(stack) > 0 {
		for {
			point = maze.Next(point)
			if point == nil {
				break
			}
			stack = append(stack, point)
		}
		i := maze.randInt() % ((len(stack) + 1) / 2)
		point = stack[i]
		stack = append(stack[:i], stack[i+1:]...)
	}
}

// Solve the maze
func (maze *Maze) Solve() {
	if maze.Solved {
		return
	}
	point := maze.Start
	stack := []*Point{point}
	solution := []*Point{point}
	visited := 1 << 12
	// Repeat until we reach the goal
	for !point.Equal(maze.Goal) {
		maze.Directions[point.X][point.Y] |= visited
		for _, direction := range Directions {
			// Push the nearest points to the stack if not been visited yet
			if maze.Directions[point.X][point.Y]&direction == direction {
				next := point.Advance(direction)
				if maze.Directions[next.X][next.Y]&visited == 0 {
					stack = append(stack, next)
				}
			}
		}
		// Pop the stack
		point = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		// We have reached to a dead end so we pop the solution
		for last := solution[len(solution)-1]; !maze.Connected(point, last); {
			solution = solution[:len(solution)-1]
			last = solution[len(solution)-1]
		}
		solution = append(solution, point)
	}
	// Fill the solution path on the maze
	for i, point := range solution {
		if i < len(solution)-1 {
			next := solution[i+1]
			for _, direction := range Directions {
				if maze.Directions[point.X][point.Y]&direction == direction {
					temp := point.Advance(direction)
					if next.X == temp.X && next.Y == temp.Y {
						maze.Directions[point.X][point.Y] |= direction << SolutionOffset
						maze.Directions[next.X][next.Y] |= Opposite[direction] << SolutionOffset
						break
					}
				}
			}
		}
	}
	maze.Solved = true
}

// Clear the solution
func (maze *Maze) Clear() {
	all := Up | Down | Left | Right
	all |= all << VisitedOffset // Do not clear the user's path
	for _, directions := range maze.Directions {
		for j := range directions {
			directions[j] &= all
		}
	}
	maze.Solved = false
}

// Move the cursor
func (maze *Maze) Move(direction int) {
	point := maze.Cursor
	next := point.Advance(direction)
	// If there's a path on the maze, we can move the cursor
	if maze.Contains(next) && maze.Directions[point.X][point.Y]&direction == direction {
		maze.Directions[point.X][point.Y] ^= direction << VisitedOffset
		maze.Directions[next.X][next.Y] ^= Opposite[direction] << VisitedOffset
		maze.Cursor = next
	}
	maze.Started = true
	// Check if the cursor has reached the goal or not
	maze.Finished = maze.Cursor.Equal(maze.Goal)
}

// Undo the visited path
func (maze *Maze) Undo() {
	point := maze.Cursor
	next := point
	for {
		// Find the previous point
		for _, direction := range Directions {
			if (maze.Directions[point.X][point.Y]>>VisitedOffset)&direction != 0 {
				next = point.Advance(direction)
				maze.Directions[point.X][point.Y] ^= direction << VisitedOffset
				maze.Directions[next.X][next.Y] ^= Opposite[direction] << VisitedOffset
				break
			}
		}
		if point.Equal(next) {
			// Previous point was not found (for example: the start point)
			break
		}
		// Move backward
		point = next
		// If there's another path which has not been visited, stop the procedure
		count := 0
		for _, direction := range Directions {
			if maze.Directions[next.X][next.Y]&direction != 0 {
				count++
			}
		}
		// The path we came from, we visited once and another
		if count > 2 {
			break
		}
	}
	// Move back the cursor
	maze.Cursor = point
	maze.Finished = maze.Cursor.Equal(maze.Goal)
}

// Format is the printing format of the maze
type Format struct {
	Wall               string
	Path               string
	StartLeft          string
	StartRight         string
	GoalLeft           string
	GoalRight          string
	Solution           string
	SolutionStartLeft  string
	SolutionStartRight string
	SolutionGoalLeft   string
	SolutionGoalRight  string
	Visited            string
	VisitedStartLeft   string
	VisitedStartRight  string
	VisitedGoalLeft    string
	VisitedGoalRight   string
	Cursor             string
}

// Default format
var Default = &Format{
	Wall:               "##",
	Path:               "  ",
	StartLeft:          "S ",
	StartRight:         " S",
	GoalLeft:           "G ",
	GoalRight:          " G",
	Solution:           "::",
	SolutionStartLeft:  "S:",
	SolutionStartRight: ":S",
	SolutionGoalLeft:   "G:",
	SolutionGoalRight:  ":G",
	Visited:            "..",
	VisitedStartLeft:   "S.",
	VisitedStartRight:  ".S",
	VisitedGoalLeft:    "G.",
	VisitedGoalRight:   ".G",
	Cursor:             "::",
}

// Color format
var Color = &Format{
	Wall:               "\x1b[7m  \x1b[0m",
	Path:               "  ",
	StartLeft:          "S ",
	StartRight:         " S",
	GoalLeft:           "G ",
	GoalRight:          " G",
	Solution:           "\x1b[44;1m  \x1b[0m",
	SolutionStartLeft:  "\x1b[44;1mS \x1b[0m",
	SolutionStartRight: "\x1b[44;1m S\x1b[0m",
	SolutionGoalLeft:   "\x1b[44;1mG \x1b[0m",
	SolutionGoalRight:  "\x1b[44;1m G\x1b[0m",
	Visited:            "\x1b[42;1m  \x1b[0m",
	VisitedStartLeft:   "\x1b[42;1mS \x1b[0m",
	VisitedStartRight:  "\x1b[42;1m S\x1b[0m",
	VisitedGoalLeft:    "\x1b[42;1mG \x1b[0m",
	VisitedGoalRight:   "\x1b[42;1m G\x1b[0m",
	Cursor:             "\x1b[43;1m  \x1b[0m",
}

func plot(img *image.RGBA, x, y, scale int, c color.Color) {
	for dy := 0; dy < scale; dy++ {
		for dx := 0; dx < scale; dx++ {
			img.Set(x*scale+dx, y*scale+dy, c)
		}
	}
}

// PrintPNG outputs the maze to the IO writer as PNG image
func (maze *Maze) PrintPNG(writer io.Writer, scale int) {
	var sb strings.Builder
	maze.Print(&sb, Default)
	lines := strings.Split(strings.TrimSpace(sb.String()), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	width := len(lines[0]) / 2
	height := len(lines)
	img := image.NewRGBA(image.Rect(0, 0, width*scale, height*scale))
	for y := 0; y < height; y++ {
		if y >= len(lines) {
			continue
		}
		for x := 0; x < width; x++ {
			if x*2 >= len(lines[y]) {
				continue
			}
			switch lines[y][x*2 : x*2+2] {
			case "##":
				plot(img, x, y, scale, color.Black)
			case "::":
				plot(img, x, y, scale, color.RGBA{0, 0, 255, 255})
			case "S ", " S", "S:", ":S":
				plot(img, x, y, scale, color.RGBA{255, 0, 0, 255})
			case "G ", " G", "G:", ":G":
				plot(img, x, y, scale, color.RGBA{0, 255, 0, 255})
			default:
				plot(img, x, y, scale, color.White)
			}
		}
	}
	png.Encode(writer, img)
}

// PrintSVG outputs the maze to the IO writer as SVG image
func (maze *Maze) PrintSVG(writer io.Writer, scale int) {
	var sb strings.Builder
	maze.Print(&sb, Default)
	lines := strings.Split(strings.TrimSpace(sb.String()), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	width := len(lines[0]) / 2
	height := len(lines)
	fmt.Fprintf(writer, `<svg viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">`+"\n", width*scale, height*scale)
	fmt.Fprintf(writer, `<rect width="%d" height="%d" fill="white" />`+"\n", width*scale, height*scale)
	for y := 0; y < height; y++ {
		if y >= len(lines) {
			continue
		}
		for x := 0; x < width; x++ {
			if x*2 >= len(lines[y]) {
				continue
			}
			switch lines[y][x*2 : x*2+2] {
			case "##":
				fmt.Fprintf(writer, `<rect x="%d" y="%d" width="%d" height="%d" fill="black" />`+"\n", x*scale, y*scale, scale, scale)
			case "::":
				fmt.Fprintf(writer, `<rect x="%d" y="%d" width="%d" height="%d" fill="blue" />`+"\n", x*scale, y*scale, scale, scale)
			case "S ", " S", "S:", ":S":
				fmt.Fprintf(writer, `<rect x="%d" y="%d" width="%d" height="%d" fill="red" />`+"\n", x*scale, y*scale, scale, scale)
			case "G ", " G", "G:", ":G":
				fmt.Fprintf(writer, `<rect x="%d" y="%d" width="%d" height="%d" fill="green" />`+"\n", x*scale, y*scale, scale, scale)
			default:
			}
		}
	}
	fmt.Fprintln(writer, "</svg>")
}

// Print out the maze to the IO writer
func (maze *Maze) Print(writer io.Writer, format *Format) {
	fmt.Fprint(writer, maze.String(format))
}

func (maze *Maze) String(format *Format) string {
	var sb strings.Builder
	// If solved or started, it changes the appearance of the start and the goal
	startLeft := format.StartLeft
	if maze.Solved {
		startLeft = format.SolutionStartLeft
	} else if maze.Started {
		startLeft = format.VisitedStartLeft
	}
	startRight := format.StartRight
	if maze.Solved {
		startRight = format.SolutionStartRight
	} else if maze.Started {
		startRight = format.VisitedStartRight
	}
	goalLeft := format.GoalLeft
	if maze.Solved {
		goalLeft = format.SolutionGoalLeft
	} else if maze.Finished {
		goalLeft = format.VisitedGoalLeft
	}
	goalRight := format.GoalRight
	if maze.Solved {
		goalRight = format.SolutionGoalRight
	} else if maze.Finished {
		goalRight = format.VisitedGoalRight
	}
	// We can use & to check if the direction is the solution path or the path user has visited
	solved := (Up | Down | Left | Right) << SolutionOffset
	visited := (Up | Down | Left | Right) << VisitedOffset
	// Print out the maze
	for x, row := range maze.Directions {
		// There are two lines printed for each maze lines
		for _, direction := range []int{Up, Right} {
			// The left wall
			if maze.Start.X == x && maze.Start.Y == 0 && direction == Right {
				sb.WriteString(startLeft)
			} else if maze.Goal.X == x && maze.Goal.Y == 0 && maze.Width > 1 && direction == Right {
				sb.WriteString(goalLeft)
			} else {
				sb.WriteString(format.Wall)
			}
			for y, directions := range row {
				// In the `direction == Right` line, we print the path cell
				if direction == Right {
					if directions&solved != 0 {
						sb.WriteString(format.Solution)
					} else if directions&visited != 0 {
						if maze.Cursor.X == x && maze.Cursor.Y == y {
							sb.WriteString(format.Cursor)
						} else {
							sb.WriteString(format.Visited)
						}
					} else {
						sb.WriteString(format.Path)
					}
				}
				// Print the start or goal point on the right hand side
				if maze.Start.X == x && maze.Start.Y == y && y == maze.Width-1 && 0 < y && direction == Right {
					sb.WriteString(startRight)
				} else if maze.Goal.X == x && maze.Goal.Y == y && y == maze.Width-1 && direction == Right {
					sb.WriteString(goalRight)
				} else
				// Print the start or goal point on the top wall of the maze
				if maze.Start.X == x && maze.Start.Y == y && x == 0 && maze.Height > 1 && 0 < y && y < maze.Width-1 && direction == Up {
					sb.WriteString(startLeft)
				} else if maze.Goal.X == x && maze.Goal.Y == y && x == 0 && maze.Height > 1 && 0 < y && y < maze.Width-1 && direction == Up {
					sb.WriteString(goalLeft)
				} else
				// If there is a path in the direction (Up or Right) on the maze
				if directions&direction != 0 {
					// Print the path cell, or the solution cell if solved or the visited cells if the user visited
					if (directions>>SolutionOffset)&direction != 0 {
						sb.WriteString(format.Solution)
					} else if (directions>>VisitedOffset)&direction != 0 {
						sb.WriteString(format.Visited)
					} else {
						sb.WriteString(format.Path)
					}
				} else {
					// Print the wall cell
					sb.WriteString(format.Wall)
				}
				// In the `direction == Up` line, we print the wall cell
				if direction == Up {
					sb.WriteString(format.Wall)
				}
			}
			sb.WriteString("\n")
		}
	}
	// Print the bottom wall of the maze
	sb.WriteString(format.Wall)
	for y := 0; y < maze.Width; y++ {
		if maze.Start.X == maze.Height-1 && maze.Start.Y == y && maze.Height > 1 && 0 < y && y < maze.Width-1 {
			sb.WriteString(startLeft)
		} else if maze.Goal.X == maze.Height-1 && maze.Goal.Y == y && 0 < y && y < maze.Width-1 {
			sb.WriteString(goalRight)
		} else {
			sb.WriteString(format.Wall)
		}
		sb.WriteString(format.Wall)
	}
	return sb.String()
}
