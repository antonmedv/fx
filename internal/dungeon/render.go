package dungeon

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	wallColor   = rgb{0.78, 0.72, 0.66}
	floorColor  = rgb{0.75, 0.7, 0.62}
	memoryColor = rgb{0.2, 0.22, 0.32}
	glow        = 0.14 // how much of the light tints the floor
)

var itemLook = map[itemKind]struct {
	glyph rune
	color rgb
}{
	potion: {'!', rgb{1, 0.3, 0.5}},
	gold:   {'$', rgb{1, 0.85, 0.2}},
	torch:  {'(', rgb{1, 0.6, 0.2}},
	weapon: {'/', rgb{0.75, 0.85, 1}},
	armor:  {'[', rgb{0.6, 0.7, 0.8}},
	brace:  {'{', rgb{1, 0.9, 0.3}},
}

func (g *game) View() tea.View {
	v := tea.NewView(g.render())
	v.AltScreen = true
	v.WindowTitle = "fx dungeon"
	return v
}

func (g *game) render() string {
	if g.width == 0 {
		return ""
	}
	return frame(g.screen(), g.width, g.height)
}

// frame paints the whole window black, whatever the terminal's own colors.
func frame(s string, width, height int) string {
	const paint = "\x1b[0;38;2;217;217;217;48;2;0;0;0m" // light gray on black
	lines := strings.Split(s, "\n")
	var sb strings.Builder
	for i := range height {
		line := ""
		if i < len(lines) {
			line = ansi.Truncate(lines[i], width, "")
		}
		sb.WriteString(paint)
		sb.WriteString(line)
		sb.WriteString(strings.Repeat(" ", width-ansi.StringWidth(line)))
		sb.WriteString("\x1b[0m")
		if i < height-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func (g *game) screen() string {
	if g.width < 40 || g.height < 12 {
		return "The dungeon needs a bigger terminal."
	}
	if g.showHelp {
		return help
	}

	viewH := g.height - 3
	ox := camera(g.p.x, g.width, g.lvl.w)
	oy := camera(g.p.y, viewH, g.lvl.h)

	var sb strings.Builder
	var p pen
	for sy := range viewH {
		for sx := range g.width {
			ch, fg, bg := g.cell(ox+sx, oy+sy)
			p.draw(&sb, ch, fg, bg)
		}
		p = pen{} // frame starts every line afresh
		sb.WriteByte('\n')
	}
	g.status(&sb, &p)

	var lines []string
	switch {
	case g.won:
		lines = []string{
			fmt.Sprintf("You won with %d gold in %d turns!", g.p.gold, g.turn),
			"Press r to play again or q to quit.",
		}
	case g.over:
		lines = []string{g.cause + fmt.Sprintf(" You had %d gold.", g.p.gold), "Press r to try again or q to quit."}
	default:
		lines = g.msgs[max(0, len(g.msgs)-2):]
	}
	for i, line := range lines {
		c := rgb{0.85, 0.85, 0.85}
		if i == 0 && len(lines) > 1 && !g.over {
			c = rgb{0.5, 0.5, 0.5} // the older message
		}
		sb.WriteByte('\n')
		p = pen{} // frame starts every line afresh
		p.text(&sb, line, c)
	}
	return sb.String()
}

// camera returns the map coordinate of the left (or top) edge of the view.
func camera(pos, view, world int) int {
	if world <= view {
		return -(view - world) / 2
	}
	return min(max(pos-view/2, 0), world-view)
}

// cell returns the glyph at (x, y) and its colors. Lit floor glows faintly
// in the color of the light falling on it.
func (g *game) cell(x, y int) (ch rune, fg, bg rgb) {
	l := g.lvl
	if !l.in(x, y) {
		return ' ', rgb{}, bg
	}
	i := l.idx(x, y)
	t := l.at(x, y)
	lum := g.light(i, g.frame)
	if g.visible[i] && lum.brightness() > threshold {
		lum = tone(lum)
		k := lum.brightness()
		if t != wall {
			bg = lum.scale(glow)
		}
		if x == g.p.x && y == g.p.y {
			return '@', rgb{1, 1, 0.9}, bg
		}
		if m := l.monsterAt(x, y); m != nil {
			return m.glyph, m.color.scale(max(k, 0.55)), bg
		}
		if it := l.itemAt(x, y); it != nil {
			look := itemLook[it.kind]
			return look.glyph, look.color.scale(max(k, 0.5)), bg
		}
		switch t {
		case wall:
			return '#', wallColor.mul(lum).add(memoryColor.scale(1 - k)), bg
		case stairs:
			return '>', rgb{1, 1, 1}.scale(max(k, 0.6)), bg
		case brazier:
			return 'Ω', rgb{1, 0.65, 0.25}.scale(0.8 + 0.2*k), bg
		case crystal:
			return '*', rgb{0.55, 0.8, 1}.scale(0.8 + 0.2*k), bg
		}
		return '.', floorColor.mul(lum).add(memoryColor.scale(1 - k)), bg
	}
	if !l.seen[i] {
		return ' ', rgb{}, bg
	}
	if it := l.itemAt(x, y); it != nil {
		return itemLook[it.kind].glyph, memoryColor, bg
	}
	switch t {
	case wall:
		return '#', memoryColor, bg
	case stairs:
		return '>', memoryColor.scale(1.6), bg
	case brazier:
		return 'Ω', memoryColor, bg
	case crystal:
		return '*', memoryColor, bg
	}
	return '.', memoryColor, bg
}

func (g *game) status(sb *strings.Builder, p *pen) {
	hpColor := rgb{0.4, 0.9, 0.4}
	switch {
	case g.p.hp*4 <= g.p.maxHP:
		hpColor = rgb{1, 0.3, 0.3}
	case g.p.hp*2 <= g.p.maxHP:
		hpColor = rgb{1, 0.8, 0.3}
	}
	gray := rgb{0.6, 0.6, 0.6}
	p.text(sb, fmt.Sprintf(" HP %d/%d ", g.p.hp, g.p.maxHP), hpColor)
	p.text(sb, bar(g.p.hp, g.p.maxHP, 10), hpColor)
	p.text(sb, fmt.Sprintf("  ATK %d  DEF %d  LV %d (%d/%d)  Torch ", g.p.atk, g.p.def, g.p.level, g.p.xp, g.p.level*10), gray)
	p.text(sb, bar(min(g.p.fuel, fullTorch), fullTorch, 5), torchColor)
	p.text(sb, fmt.Sprintf("  $%d  Depth %d/%d", g.p.gold, g.depth, maxDepth), gray)
}

func bar(v, total, width int) string {
	n := 0
	if total > 0 {
		n = (v*width + total - 1) / total
	}
	n = min(max(n, 0), width)
	return strings.Repeat("█", n) + strings.Repeat("░", width-n)
}

// pen writes truecolor escape codes only when the colors change. Its zero
// value matches the start of a line painted by frame: black background.
type pen struct {
	fg, bg [3]uint8
	hasFg  bool
}

func to8(c rgb) [3]uint8 {
	b := func(v float64) uint8 { return uint8(min(max(v, 0), 1)*255 + 0.5) }
	return [3]uint8{b(c.r), b(c.g), b(c.b)}
}

func (p *pen) color(sb *strings.Builder, fg rgb) {
	if c := to8(fg); !p.hasFg || c != p.fg {
		fmt.Fprintf(sb, "\x1b[38;2;%d;%d;%dm", c[0], c[1], c[2])
		p.fg, p.hasFg = c, true
	}
}

func (p *pen) background(sb *strings.Builder, bg rgb) {
	if c := to8(bg); c != p.bg {
		fmt.Fprintf(sb, "\x1b[48;2;%d;%d;%dm", c[0], c[1], c[2])
		p.bg = c
	}
}

func (p *pen) draw(sb *strings.Builder, ch rune, fg, bg rgb) {
	p.background(sb, bg)
	if ch != ' ' {
		p.color(sb, fg)
	}
	sb.WriteRune(ch)
}

func (p *pen) text(sb *strings.Builder, s string, fg rgb) {
	p.background(sb, rgb{})
	p.color(sb, fg)
	sb.WriteString(s)
}

const help = `
  fx dungeon

  Find the Golden Brace { on depth 8 and escape the Stack Overflow dragon.

  Move        arrows, hjkl, wasd; yubn for diagonals
  Attack      walk into a monster
  Wait        . or space
  Descend     > on stairs
  Quit        q or esc

  @  you          >  stairs         !  potion      $  gold
  (  torch        /  weapon         [  armor       {  the Golden Brace
  Ω  brazier      *  crystal

  Your torch burns down as you walk; its light shrinks with it.
  Monsters in the dark stay hidden until light falls on them.

  Press any key to return.
`
