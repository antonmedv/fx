// Package dungeon is a small roguelike hidden in fx: fx --dungeon.
package dungeon

import (
	"fmt"
	"math/rand/v2"
	"time"

	tea "charm.land/bubbletea/v2"
)

type kind struct {
	name               string
	glyph              rune
	color              rgb
	hp, atk, def, xp   int
	minDepth, maxDepth int
}

var bestiary = []*kind{
	{"rat", 'r', rgb{0.7, 0.55, 0.4}, 3, 2, 0, 1, 1, 3},
	{"null", 'n', rgb{0.75, 0.75, 0.85}, 4, 2, 0, 2, 1, 4},
	{"goblin", 'g', rgb{0.4, 0.85, 0.3}, 7, 3, 1, 3, 2, 6},
	{"undefined", 'u', rgb{0.8, 0.4, 0.9}, 9, 4, 1, 5, 3, 7},
	{"orc", 'o', rgb{0.3, 0.7, 0.35}, 13, 5, 2, 7, 4, 8},
	{"NaN", 'N', rgb{1, 0.35, 0.35}, 15, 6, 2, 9, 5, 8},
	{"troll", 'T', rgb{0.55, 0.75, 0.4}, 24, 7, 3, 14, 6, 8},
}

var dragon = &kind{"Stack Overflow dragon", 'D', rgb{1, 0.3, 0.1}, 45, 9, 4, 50, maxDepth, maxDepth}

type monster struct {
	*kind
	x, y, hp int
	awake    bool
}

type itemKind int

const (
	potion itemKind = iota
	gold
	torch
	weapon
	armor
	brace
)

type item struct {
	kind   itemKind
	x, y   int
	amount int
}

type player struct {
	x, y       int
	hp, maxHP  int
	atk, def   int
	level, xp  int
	gold, fuel int
}

var torchColor = rgb{1, 0.72, 0.42}

const (
	sightRadius = 60
	maxFuel     = 900
	fullTorch   = 500 // fuel at which the torch burns at full radius
	threshold   = 0.035
)

type tickMsg struct{}

type game struct {
	rng   *rand.Rand
	lvl   *level
	depth int
	p     player
	turn  int
	frame int
	msgs  []string

	over, won bool
	cause     string
	showHelp  bool

	width, height int

	visible []bool    // the player's line of sight
	torch   []float64 // falloff of the player's torch
}

func newGame(seed uint64) *game {
	g := &game{
		rng: rand.New(rand.NewPCG(seed, 0xf00d)),
		p:   player{hp: 20, maxHP: 20, atk: 4, def: 1, level: 1, fuel: fullTorch},
	}
	g.log("You enter the dungeon beneath fx. Legends speak of the Golden Brace { on depth %d.", maxDepth)
	g.log("Press ? for help.")
	g.descend()
	return g
}

// Run starts the game.
func Run() error {
	_, err := tea.NewProgram(newGame(uint64(time.Now().UnixNano()))).Run()
	return err
}

func (g *game) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func (g *game) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.width, g.height = msg.Width, msg.Height
	case tickMsg:
		g.frame++
		return g, tick()
	case tea.KeyPressMsg:
		return g, g.key(msg.String())
	}
	return g, nil
}

var moves = map[string][2]int{
	"h": {-1, 0}, "left": {-1, 0}, "a": {-1, 0},
	"l": {1, 0}, "right": {1, 0}, "d": {1, 0},
	"k": {0, -1}, "up": {0, -1}, "w": {0, -1},
	"j": {0, 1}, "down": {0, 1}, "s": {0, 1},
	"y": {-1, -1}, "u": {1, -1}, "b": {-1, 1}, "n": {1, 1},
}

func (g *game) key(k string) tea.Cmd {
	if k == "ctrl+c" || k == "esc" || k == "q" {
		return tea.Quit
	}
	if g.showHelp {
		g.showHelp = false
		return nil
	}
	if g.over {
		if k == "r" {
			w, h := g.width, g.height
			*g = *newGame(g.rng.Uint64())
			g.width, g.height = w, h
		}
		return nil
	}
	if d, ok := moves[k]; ok {
		g.move(d[0], d[1])
		return nil
	}
	switch k {
	case ".", "space", "z":
		g.endTurn()
	case ">":
		if g.lvl.at(g.p.x, g.p.y) == stairs {
			g.descend()
		} else {
			g.log("There are no stairs here.")
		}
	case "?":
		g.showHelp = true
	}
	return nil
}

func (g *game) log(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	if n := len(g.msgs); n > 0 && g.msgs[n-1] == msg {
		return
	}
	g.msgs = append(g.msgs, msg)
	if len(g.msgs) > 50 {
		g.msgs = g.msgs[len(g.msgs)-50:]
	}
}

func (g *game) descend() {
	g.depth++
	g.lvl = generate(g.rng, g.depth)
	g.p.x, g.p.y = g.lvl.startX, g.lvl.startY
	g.visible = make([]bool, g.lvl.w*g.lvl.h)
	g.torch = make([]float64, g.lvl.w*g.lvl.h)
	if g.depth > 1 {
		g.log("You descend to depth %d. The air grows colder.", g.depth)
	}
	g.look()
}

func (g *game) move(dx, dy int) {
	nx, ny := g.p.x+dx, g.p.y+dy
	if m := g.lvl.monsterAt(nx, ny); m != nil {
		g.attack(m)
		g.endTurn()
		return
	}
	switch t := g.lvl.at(nx, ny); t {
	case brazier:
		g.log("The brazier is too hot to touch.")
		return
	case crystal:
		g.log("The crystal hums softly.")
		return
	case wall:
		return // bumping into walls is free
	case stairs:
		g.log("A staircase leads down. Press > to descend.")
	}
	g.p.x, g.p.y = nx, ny
	g.pickup()
	g.endTurn()
}

func (g *game) pickup() {
	it := g.lvl.itemAt(g.p.x, g.p.y)
	if it == nil {
		return
	}
	switch it.kind {
	case potion:
		heal := 8 + g.depth
		g.p.hp = min(g.p.maxHP, g.p.hp+heal)
		g.log("You drink a potion. You feel better.")
	case gold:
		g.p.gold += it.amount
		g.log("You pick up %d gold.", it.amount)
	case torch:
		g.p.fuel = min(maxFuel, g.p.fuel+300)
		g.log("You light a fresh torch.")
	case weapon:
		g.p.atk++
		g.log("You find a sharper blade. ATK +1.")
	case armor:
		g.p.def++
		g.log("You strap on a piece of armor. DEF +1.")
	case brace:
		g.over, g.won = true, true
		g.log("You lift the Golden Brace {. The dungeon is well-formed at last!")
	}
	g.lvl.items = remove(g.lvl.items, it)
}

func (g *game) roll(atk, def int) int {
	return max(0, 1+g.rng.IntN(max(atk, 1))-g.rng.IntN(def+1))
}

func (g *game) attack(m *monster) {
	m.awake = true
	d := g.roll(g.p.atk, m.def)
	if d == 0 {
		g.log("You miss the %s.", m.name)
		return
	}
	m.hp -= d
	if m.hp > 0 {
		g.log("You hit the %s for %d.", m.name, d)
		return
	}
	g.log("You slay the %s.", m.name)
	g.lvl.monsters = remove(g.lvl.monsters, m)
	g.p.xp += m.xp
	for g.p.xp >= g.p.level*10 {
		g.p.xp -= g.p.level * 10
		g.p.level++
		g.p.maxHP += 5
		g.p.hp = g.p.maxHP
		g.p.atk++
		if g.p.level%2 == 1 {
			g.p.def++
		}
		g.log("You feel stronger! Welcome to level %d.", g.p.level)
	}
}

func (g *game) endTurn() {
	if g.over {
		return // nothing moves after the Golden Brace is taken
	}
	g.turn++
	if g.p.fuel > 0 {
		g.p.fuel--
		switch g.p.fuel {
		case 100:
			g.log("Your torch sputters. Find another soon.")
		case 0:
			g.log("Your torch burns out. Only embers remain.")
		}
	}
	if g.turn%12 == 0 && g.p.hp < g.p.maxHP {
		g.p.hp++
	}
	g.look()
	g.monstersAct()
}

func torchRadius(fuel int) int {
	if fuel <= 0 {
		return 2
	}
	return 3 + min(fuel, fullTorch)/100
}

// look recomputes what the player sees and remembers what is lit.
func (g *game) look() {
	l := g.lvl
	clear(g.torch)
	r := torchRadius(g.p.fuel)
	l.fov(g.p.x, g.p.y, r, func(x, y int, d float64) {
		g.torch[l.idx(x, y)] = falloff(d, r)
	})
	clear(g.visible)
	l.fov(g.p.x, g.p.y, sightRadius, func(x, y int, _ float64) {
		i := l.idx(x, y)
		g.visible[i] = true
		if g.light(i, -1).brightness() > threshold {
			l.seen[i] = true
		}
	})
}

// light sums every light source at cell i on animation frame t; a negative
// frame means no flicker.
func (g *game) light(i, t int) rgb {
	var c rgb
	for _, li := range g.lvl.lights {
		if f := li.lum[i]; f > 0 {
			k := 1.0
			if t >= 0 {
				k = li.intensity(t)
			}
			c = c.add(li.color.scale(f * k))
		}
	}
	if f := g.torch[i]; f > 0 {
		k := 1.0
		if t >= 0 {
			k = 1 + 0.08*(fastNoise(t)-0.5)
		}
		c = c.add(torchColor.scale(f * k))
	}
	return c
}

// fastNoise is a cheap deterministic value in [0, 1) for torch flicker.
func fastNoise(t int) float64 {
	x := uint32(t)*2654435761 + 0x9e3779b9
	x ^= x >> 15
	x *= 0x85ebca6b
	x ^= x >> 13
	return float64(x%1000) / 1000
}

func (g *game) lit(i int) bool {
	return g.visible[i] && g.light(i, -1).brightness() > threshold
}

func (g *game) monstersAct() {
	l := g.lvl
	dist := l.distances(g.p.x, g.p.y)
	for _, m := range l.monsters {
		i := l.idx(m.x, m.y)
		if !m.awake {
			if g.visible[i] && dist[i] >= 0 && dist[i] <= 9 && g.rng.IntN(3) > 0 {
				m.awake = true
				if g.lit(i) {
					g.log("The %s notices you.", m.name)
				}
			}
			continue
		}
		if max(abs(m.x-g.p.x), abs(m.y-g.p.y)) == 1 {
			d := g.roll(m.atk, g.p.def)
			if d == 0 {
				g.log("The %s misses.", m.name)
				continue
			}
			g.p.hp -= d
			g.log("The %s hits you for %d.", m.name, d)
			if g.p.hp <= 0 {
				g.p.hp = 0
				g.over = true
				g.cause = fmt.Sprintf("Slain by the %s on depth %d.", m.name, g.depth)
				return
			}
			continue
		}
		best, bx, by := dist[i], m.x, m.y
		for _, d := range dirs8 {
			nx, ny := m.x+d[0], m.y+d[1]
			if !l.at(nx, ny).passable() || l.monsterAt(nx, ny) != nil {
				continue
			}
			if j := l.idx(nx, ny); dist[j] >= 0 && (best < 0 || dist[j] < best) {
				best, bx, by = dist[j], nx, ny
			}
		}
		m.x, m.y = bx, by
	}
}

func remove[T comparable](s []T, v T) []T {
	for i, x := range s {
		if x == v {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
