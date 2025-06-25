package maze

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type mob struct {
	x, y             float64 // exact position
	targetX, targetY int     // integer cell coords we’re heading toward
	speed            float64 // base speed
	color            string
}

type model struct {
	// Maze size.
	size int

	// labyrinth: '#', 'S', 'G' indicates a wall, ' ' empty space.
	labyrinth [][]rune

	// Player state.
	playerX, playerY float64
	playerAngle      float64

	// Game board dimensions (cells).
	width, height int

	// Terminal dimensions.
	termWidth, termHeight int

	mobs []mob

	// Virtual canvas (each game cell becomes 2×4 pixels).
	canvas [][]bool

	// Cells (the final board; dimensions: height x width).
	cells [][]string

	// Overlay distance per cell (for occlusion).
	overlayDist [][]float64

	// Wall depth per cell column.
	wallDepthCell []float64

	// Wall's color.
	color []string
}

func createModel(size int) *model {
	maze := NewMaze(size, size)
	maze.Generate()

	lines := strings.Split(maze.String(Default), "\n")
	lab := make([][]rune, len(lines))
	for i, line := range lines {
		lab[i] = []rune(line)
	}

	var mobs []mob
	if size >= 0 {
		// Find all valid walkable positions
		var positions []struct{ x, y float64 }
		for y := range lab {
			for x := range lab[y] {
				if lab[y][x] == ' ' {
					positions = append(positions, struct{ x, y float64 }{
						x: float64(x) + 0.5,
						y: float64(y) + 0.5,
					})
				}
			}
		}

		// Shuffle and pick first 5 positions
		rand.Shuffle(len(positions), func(i, j int) {
			positions[i], positions[j] = positions[j], positions[i]
		})

		// Add mobs at random positions with random velocities
		for i := 0; i < 5 && i < len(positions); i++ {
			start := positions[i]
			// pick a random neighbor cell as first target:
			gx, gy := int(start.x), int(start.y)
			mobs = append(mobs, mob{
				x:       start.x,
				y:       start.y,
				targetX: gx,
				targetY: gy,
				speed:   0.01 + rand.Float64()*0.01, // vary speeds
				color:   fmt.Sprintf("%d", i+1),
			})
		}
	}

	m := &model{
		size:        size,
		labyrinth:   lab,
		playerX:     1.5,
		playerY:     1.5,
		playerAngle: 0,
		width:       80,
		height:      24,
		termWidth:   80,
		termHeight:  24,
		mobs:        mobs,
	}
	m.allocateBuffers()
	return m
}

// tickMsg triggers periodic updates.
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) Init() tea.Cmd {
	return tickCmd()
}

// allocateBuffers (re)allocates the reusable slices according to the board dimensions.
func (m *model) allocateBuffers() {
	if m.width == 0 || m.height == 0 {
		return
	}

	virtualWidth := m.width * 2
	virtualHeight := m.height * 4

	m.canvas = make([][]bool, virtualHeight)
	for i := 0; i < virtualHeight; i++ {
		m.canvas[i] = make([]bool, virtualWidth)
	}

	m.color = make([]string, virtualWidth)

	m.cells = make([][]string, m.height)
	for i := 0; i < m.height; i++ {
		m.cells[i] = make([]string, m.width)
	}

	m.overlayDist = make([][]float64, m.height)
	for i := 0; i < m.height; i++ {
		m.overlayDist[i] = make([]float64, m.width)
		for j := 0; j < m.width; j++ {
			m.overlayDist[i][j] = 17 // maxDepth (16) + 1
		}
	}

	m.wallDepthCell = make([]float64, m.width)
}

func (m *model) walkableNeighbors(gx, gy int) []struct{ x, y int } {
	var out []struct{ x, y int }
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := gx+d[0], gy+d[1]
		if !m.isWall(float64(nx)+0.5, float64(ny)+0.5) {
			out = append(out, struct{ x, y int }{nx, ny})
		}
	}
	return out
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Update terminal dimensions and clamp board size.
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.width = min(msg.Width-4, 120)
		m.height = min(msg.Height-4, 34)
		m.allocateBuffers()
		return m, nil

	case tickMsg:
		// Update mob positions with simple bouncing.
		for i := range m.mobs {
			mb := &m.mobs[i]

			// If we’re at (or very near) the target cell center, choose a new neighbor:
			centerX := float64(mb.targetX) + 0.5
			centerY := float64(mb.targetY) + 0.5
			if math.Hypot(mb.x-centerX, mb.y-centerY) < 0.1 {
				nbrs := m.walkableNeighbors(mb.targetX, mb.targetY)
				if len(nbrs) > 0 {
					choice := nbrs[rand.IntN(len(nbrs))]
					mb.targetX, mb.targetY = choice.x, choice.y
				}
			}

			// Compute velocity toward target (plus a tiny random jitter)
			dx := centerX - mb.x
			dy := centerY - mb.y
			dist := math.Hypot(dx, dy)
			if dist > 0 {
				vx := dx / dist * mb.speed
				vy := dy / dist * mb.speed
				// stumbling: 2% chance to wiggle speed slightly
				if rand.Float64() < 0.02 {
					factor := 0.7 + rand.Float64()*0.6
					vx *= factor
					vy *= factor
				}
				// attempt move, but bounce‐off triggers a re‐target
				if m.isWall(mb.x+vx, mb.y) {
					mb.targetX, mb.targetY = int(mb.x), int(mb.y)
				} else {
					mb.x += vx
				}
				if m.isWall(mb.x, mb.y+vy) {
					mb.targetX, mb.targetY = int(mb.x), int(mb.y)
				} else {
					mb.y += vy
				}
			}
		}
		return m, tickCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left":
			m.playerAngle -= 0.1
		case "right":
			m.playerAngle += 0.1
		case "up":
			speed := 0.2
			dx := math.Cos(m.playerAngle) * speed
			dy := math.Sin(m.playerAngle) * speed
			if !m.isWallNoG(m.playerX+dx, m.playerY) {
				m.playerX += dx
			}
			if !m.isWallNoG(m.playerX, m.playerY+dy) {
				m.playerY += dy
			}
		case "down":
			speed := 0.2
			dx := -math.Cos(m.playerAngle) * speed
			dy := -math.Sin(m.playerAngle) * speed
			if !m.isWallNoG(m.playerX+dx, m.playerY) {
				m.playerX += dx
			}
			if !m.isWallNoG(m.playerX, m.playerY+dy) {
				m.playerY += dy
			}
		}
		ch := m.at(m.playerX, m.playerY)
		if ch == 'G' {
			nm := createModel(m.size + 1)
			nm.termWidth = m.termWidth
			nm.termHeight = m.termHeight
			nm.width = m.width
			nm.height = m.height
			nm.allocateBuffers()
			return nm, nil
		}
		return m, nil
	}
	return m, nil
}

func (m *model) at(x, y float64) rune {
	gridX := int(x)
	gridY := int(y)
	if gridY < 0 || gridY >= len(m.labyrinth) || gridX < 0 || gridX >= len(m.labyrinth[0]) {
		return ' '
	}
	return m.labyrinth[gridY][gridX]
}

func (m *model) isWall(x, y float64) bool {
	gridX := int(x)
	gridY := int(y)
	if gridY < 0 || gridY >= len(m.labyrinth) || gridX < 0 || gridX >= len(m.labyrinth[0]) {
		return true
	}
	ch := m.labyrinth[gridY][gridX]
	return ch == '#' || ch == 'S' || ch == 'G'
}

func (m *model) isWallNoG(x, y float64) bool {
	gridX := int(x)
	gridY := int(y)
	if gridY < 0 || gridY >= len(m.labyrinth) || gridX < 0 || gridX >= len(m.labyrinth[0]) {
		return true
	}
	ch := m.labyrinth[gridY][gridX]
	return ch == '#' || ch == 'S'
}

func getSphereSymbol(norm float64) string {
	if norm < 0.33 {
		return "●"
	} else if norm < 0.66 {
		return "◉"
	}
	return "○"
}

func ansiColor(color string, text string) string {
	if color == "" {
		return text
	}
	return fmt.Sprintf("\033[38;5;%sm%s\033[0m", color, text)
}

func (m *model) View() string {
	// Virtual canvas resolution.
	virtualWidth := m.width * 2
	virtualHeight := m.height * 4
	fov := math.Pi / 3
	maxDepth := 100.0

	// --- Fill the virtual canvas using raycasting ---
	for x := 0; x < virtualWidth; x++ {
		rayAngle := m.playerAngle - fov/2 + fov*(float64(x)/float64(virtualWidth))
		distance := 0.0
		hitWall := false
		stepSize := 0.05
		for distance < maxDepth && !hitWall {
			distance += stepSize
			testX := m.playerX + math.Cos(rayAngle)*distance
			testY := m.playerY + math.Sin(rayAngle)*distance
			if testX < 0 || testX >= float64(len(m.labyrinth[0])) ||
				testY < 0 || testY >= float64(len(m.labyrinth)) {
				hitWall = true
				distance = maxDepth
			} else if m.isWall(testX, testY) {
				hitWall = true
				ch := m.at(testX, testY)
				if ch == 'S' {
					m.color[x] = "2" // green
				} else if ch == 'G' {
					m.color[x] = "5" // magenta
				} else {
					m.color[x] = ""
				}
			}
		}
		distance *= math.Cos(rayAngle - m.playerAngle)
		if distance < 0.0001 {
			distance = 0.0001
		}
		wallHeight := int(float64(virtualHeight) / distance)
		if wallHeight > virtualHeight {
			wallHeight = virtualHeight
		}
		wallTop := (virtualHeight - wallHeight) / 2
		wallBottom := wallTop + wallHeight
		for y := 0; y < virtualHeight; y++ {
			m.canvas[y][x] = y >= wallTop && y < wallBottom
		}
	}

	// --- Build cells from the canvas ---
	for r := 0; r < m.height; r++ {
		baseY := r * 4
		for c := 0; c < m.width; c++ {
			baseX := c * 2
			pattern := 0
			if m.canvas[baseY+0][baseX+0] {
				pattern |= 1
			}
			if m.canvas[baseY+1][baseX+0] {
				pattern |= 2
			}
			if m.canvas[baseY+2][baseX+0] {
				pattern |= 4
			}
			if m.canvas[baseY+0][baseX+1] {
				pattern |= 8
			}
			if m.canvas[baseY+1][baseX+1] {
				pattern |= 16
			}
			if m.canvas[baseY+2][baseX+1] {
				pattern |= 32
			}
			if m.canvas[baseY+3][baseX+0] {
				pattern |= 64
			}
			if m.canvas[baseY+3][baseX+1] {
				pattern |= 128
			}
			dots := string(rune(0x2800 + pattern))
			m.cells[r][c] = ansiColor(m.color[baseX], dots)
		}
	}

	// --- Compute wall depth per cell column for occlusion ---
	for c := 0; c < m.width; c++ {
		virtualX := float64(c*2 + 1)
		rayAngle := m.playerAngle - fov/2 + fov*(virtualX/float64(virtualWidth))
		distance := 0.0
		hitWall := false
		stepSize := 0.05
		for distance < maxDepth && !hitWall {
			distance += stepSize
			testX := m.playerX + math.Cos(rayAngle)*distance
			testY := m.playerY + math.Sin(rayAngle)*distance
			if testX < 0 || testX >= float64(len(m.labyrinth[0])) ||
				testY < 0 || testY >= float64(len(m.labyrinth)) {
				hitWall = true
				distance = maxDepth
			} else if m.isWall(testX, testY) {
				hitWall = true
			}
		}
		distance *= math.Cos(rayAngle - m.playerAngle)
		if distance < 0.0001 {
			distance = 0.0001
		}
		m.wallDepthCell[c] = distance
	}

	// --- Reset overlayDist ---
	for r := 0; r < m.height; r++ {
		for c := 0; c < m.width; c++ {
			m.overlayDist[r][c] = maxDepth + 1
		}
	}

	// --- Overlay mobs as spheres ---
	for _, mb := range m.mobs {
		dx := mb.x - m.playerX
		dy := mb.y - m.playerY
		distance := math.Hypot(dx, dy)
		angleToMob := math.Atan2(dy, dx)
		relativeAngle := angleToMob - m.playerAngle
		for relativeAngle > math.Pi {
			relativeAngle -= 2 * math.Pi
		}
		for relativeAngle < -math.Pi {
			relativeAngle += 2 * math.Pi
		}
		projX := int(((relativeAngle + fov/2) / fov) * float64(m.width))
		projY := m.height / 2

		mobCellSize := int((float64(m.height) * 2) / math.Pow(distance, 1.9))
		if mobCellSize < 2 {
			mobCellSize = 2
		}
		if mobCellSize > 1000 {
			mobCellSize = 1000
		}
		radius := float64(mobCellSize) / 2.0

		if float64(projX)+radius < 0 || float64(projX)-radius >= float64(m.width) {
			continue
		}
		if float64(projY)+radius < 0 || float64(projY)-radius >= float64(m.height) {
			continue
		}

		startRow := int(math.Floor(float64(projY) - radius))
		endRow := int(math.Ceil(float64(projY) + radius))
		startCol := int(math.Floor(float64(projX) - radius))
		endCol := int(math.Ceil(float64(projX) + radius))

		clampedStartRow := max(startRow, 0)
		clampedEndRow := min(endRow, m.height-1)
		clampedStartCol := max(startCol, 0)
		clampedEndCol := min(endCol, m.width-1)

		for r := clampedStartRow; r <= clampedEndRow; r++ {
			for c := clampedStartCol; c <= clampedEndCol; c++ {
				rdx := (float64(c) + 0.5) - float64(projX)
				rdy := (float64(r) + 0.5) - float64(projY)
				distCell := math.Hypot(rdx, rdy)
				if distCell > radius {
					continue
				}
				norm := distCell / radius
				symbol := getSphereSymbol(norm)
				if distance < m.wallDepthCell[c] && distance < m.overlayDist[r][c] {
					m.cells[r][c] = ansiColor(mb.color, symbol)
					m.overlayDist[r][c] = distance
				}
			}
		}
	}

	// --- Assemble final board using strings.Builder ---
	var sb strings.Builder
	for r := 0; r < m.height; r++ {
		for c := 0; c < m.width; c++ {
			sb.WriteString(m.cells[r][c])
		}
		if r < m.height-1 {
			sb.WriteByte('\n')
		}
	}
	board := sb.String()

	// --- Add border manually ---
	topLeft, topRight, bottomLeft, bottomRight := "┌", "┐", "└", "┘"
	horizontal, vertical := "─", "│"
	boardLines := strings.Split(board, "\n")
	var borderSb strings.Builder
	// Top border.
	borderSb.WriteString(topLeft)
	for i := 0; i < m.width; i++ {
		borderSb.WriteString(horizontal)
	}
	borderSb.WriteString(topRight + "\n")
	// Middle lines.
	for _, line := range boardLines {
		borderSb.WriteString(vertical + line + vertical + "\n")
	}
	// Bottom border.
	borderSb.WriteString(bottomLeft)
	for i := 0; i < m.width; i++ {
		borderSb.WriteString(horizontal)
	}
	borderSb.WriteString(bottomRight)
	borderedBoard := borderSb.String()

	// --- Center the board manually ---
	boardLines = strings.Split(borderedBoard, "\n")
	boardHeight := len(boardLines)
	boardWidth := m.width + 2 // 2 extra columns for borders.
	leftPad := (m.termWidth - boardWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (m.termHeight - boardHeight) / 2
	if topPad < 0 {
		topPad = 0
	}
	var finalSb strings.Builder
	for i := 0; i < topPad; i++ {
		finalSb.WriteString("\n")
	}
	pad := strings.Repeat(" ", leftPad)
	for _, line := range boardLines {
		finalSb.WriteString(pad + line + "\n")
	}
	return finalSb.String()
}

func Run() {
	m := createModel(3)
	program := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		panic(err)
	}
}
