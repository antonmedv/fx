package dungeon

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

var (
	wallColor   = rgb{0.78, 0.72, 0.66}
	floorColor  = rgb{0.75, 0.7, 0.62}
	memoryColor = rgb{0.2, 0.22, 0.32}
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
			ch, fg := g.cell(ox+sx, oy+sy)
			p.draw(&sb, ch, fg)
		}
		p.reset(&sb)
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
		p.text(&sb, truncate(line, g.width), c)
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

func (g *game) cell(x, y int) (rune, rgb) {
	l := g.lvl
	if !l.in(x, y) {
		return ' ', rgb{}
	}
	i := l.idx(x, y)
	t := l.at(x, y)
	lum := g.light(i, g.frame)
	if g.visible[i] && lum.brightness() > threshold {
		lum = tone(lum)
		k := lum.brightness()
		if x == g.p.x && y == g.p.y {
			return '@', rgb{1, 1, 0.9}
		}
		if m := l.monsterAt(x, y); m != nil {
			return m.glyph, m.color.scale(max(k, 0.55))
		}
		if it := l.itemAt(x, y); it != nil {
			look := itemLook[it.kind]
			return look.glyph, look.color.scale(max(k, 0.5))
		}
		switch t {
		case wall:
			return '#', wallColor.mul(lum).add(memoryColor.scale(1 - k))
		case stairs:
			return '>', rgb{1, 1, 1}.scale(max(k, 0.6))
		case brazier:
			return 'Ω', rgb{1, 0.65, 0.25}.scale(0.8 + 0.2*k)
		case crystal:
			return '*', rgb{0.55, 0.8, 1}.scale(0.8 + 0.2*k)
		}
		return '.', floorColor.mul(lum).add(memoryColor.scale(1 - k))
	}
	if !l.seen[i] {
		return ' ', rgb{}
	}
	if it := l.itemAt(x, y); it != nil {
		return itemLook[it.kind].glyph, memoryColor
	}
	switch t {
	case wall:
		return '#', memoryColor
	case stairs:
		return '>', memoryColor.scale(1.6)
	case brazier:
		return 'Ω', memoryColor
	case crystal:
		return '*', memoryColor
	}
	return '.', memoryColor
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

func truncate(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		return string(r[:width])
	}
	return s
}

// pen writes truecolor escape codes only when the color changes.
type pen struct {
	fg    [3]uint8
	dirty bool // fg is set
}

func to8(c rgb) [3]uint8 {
	b := func(v float64) uint8 { return uint8(min(max(v, 0), 1)*255 + 0.5) }
	return [3]uint8{b(c.r), b(c.g), b(c.b)}
}

func (p *pen) color(sb *strings.Builder, fg rgb) {
	if c := to8(fg); !p.dirty || c != p.fg {
		fmt.Fprintf(sb, "\x1b[38;2;%d;%d;%dm", c[0], c[1], c[2])
		p.fg, p.dirty = c, true
	}
}

func (p *pen) draw(sb *strings.Builder, ch rune, fg rgb) {
	if ch != ' ' {
		p.color(sb, fg)
	}
	sb.WriteRune(ch)
}

func (p *pen) text(sb *strings.Builder, s string, fg rgb) {
	p.color(sb, fg)
	sb.WriteString(s)
}

func (p *pen) reset(sb *strings.Builder) {
	if p.dirty {
		sb.WriteString("\x1b[0m")
	}
	*p = pen{}
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
