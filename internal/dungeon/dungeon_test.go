package dungeon

import (
	"math/rand/v2"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

func TestGenerateIsConnected(t *testing.T) {
	for seed := range uint64(50) {
		r := rand.New(rand.NewPCG(seed, 1))
		for depth := 1; depth <= maxDepth; depth++ {
			l := generate(r, depth)
			require.True(t, l.at(l.startX, l.startY).passable())
			require.Equal(t, l.passableCount(), l.reachable(l.startX, l.startY), "seed %d depth %d", seed, depth)
			for _, m := range l.monsters {
				require.True(t, l.at(m.x, m.y).passable())
			}
			for _, it := range l.items {
				require.True(t, l.at(it.x, it.y).passable())
			}
			for _, li := range l.lights {
				require.Len(t, li.lum, l.w*l.h)
			}
			hasBrace := false
			for _, it := range l.items {
				hasBrace = hasBrace || it.kind == brace
			}
			hasStairs := strings.ContainsRune(string(tilesAsRunes(l)), '>')
			if depth == maxDepth {
				require.True(t, hasBrace)
				require.False(t, hasStairs)
			} else {
				require.True(t, hasStairs)
				require.False(t, hasBrace)
			}
		}
	}
}

func tilesAsRunes(l *level) []rune {
	out := make([]rune, len(l.tiles))
	for i, t := range l.tiles {
		out[i] = map[tile]rune{wall: '#', floor: '.', stairs: '>', brazier: 'O', crystal: '*'}[t]
	}
	return out
}

// parse builds a level from a picture.
func parse(rows ...string) *level {
	l := newLevel(len(rows[0]), len(rows))
	for y, row := range rows {
		for x, c := range row {
			switch c {
			case '.':
				l.set(x, y, floor)
			case 'O':
				l.set(x, y, brazier)
			}
		}
	}
	return l
}

func visibleSet(l *level, x, y, r int) map[[2]int]bool {
	seen := map[[2]int]bool{}
	l.fov(x, y, r, func(x, y int, _ float64) {
		key := [2]int{x, y}
		if seen[key] {
			panic("visited twice")
		}
		seen[key] = true
	})
	return seen
}

func TestFovSeesWholeRoomAndItsWalls(t *testing.T) {
	l := parse(
		"###########",
		"#.........#",
		"#.........#",
		"#.........#",
		"#.........#",
		"#.........#",
		"###########",
	)
	seen := visibleSet(l, 5, 3, 20)
	require.Len(t, seen, l.w*l.h)
}

func TestFovWallsBlock(t *testing.T) {
	l := parse(
		"#########",
		"#...#...#",
		"#...#...#",
		"#...#...#",
		"#########",
	)
	seen := visibleSet(l, 2, 2, 20)
	require.True(t, seen[[2]int{4, 2}], "the wall itself")
	for y := 1; y <= 3; y++ {
		for x := 5; x <= 7; x++ {
			require.False(t, seen[[2]int{x, y}], "behind the wall at %d,%d", x, y)
		}
	}
}

func TestFovRadius(t *testing.T) {
	l := parse(
		"###########",
		"#.........#",
		"###########",
	)
	seen := visibleSet(l, 1, 1, 3)
	require.True(t, seen[[2]int{4, 1}])
	require.False(t, seen[[2]int{5, 1}])
}

func TestBrazierLightStaysInItsRoom(t *testing.T) {
	l := parse(
		"#########",
		"#O..#...#",
		"#...#...#",
		"#########",
	)
	li := &light{x: 1, y: 1, color: rgb{1, 1, 1}, radius: 7}
	l.bake(li)
	require.Equal(t, 1.0, li.lum[l.idx(1, 1)])
	require.Greater(t, li.lum[l.idx(3, 2)], 0.0)
	require.Less(t, li.lum[l.idx(3, 2)], li.lum[l.idx(2, 1)]) // dimmer farther away
	require.Zero(t, li.lum[l.idx(6, 2)])
}

func TestTorchShrinksWithFuel(t *testing.T) {
	require.Equal(t, 8, torchRadius(fullTorch))
	require.Equal(t, 8, torchRadius(maxFuel))
	require.Equal(t, 3, torchRadius(1))
	require.Equal(t, 2, torchRadius(0))
}

func press(g *game, k string) {
	var msg tea.KeyPressMsg
	switch k {
	case "left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft}
	default:
		msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
	}
	g.Update(msg)
}

func TestBumpingWallIsFree(t *testing.T) {
	g := newGame(1)
	for g.lvl.at(g.p.x-1, g.p.y) != wall {
		g.p.x--
	}
	g.lvl.monsters = nil
	turn := g.turn
	press(g, "left")
	require.Equal(t, turn, g.turn)
	press(g, ".")
	require.Equal(t, turn+1, g.turn)
}

func TestDescend(t *testing.T) {
	g := newGame(2)
	press(g, ">")
	require.Equal(t, 1, g.depth)
	for i, tl := range g.lvl.tiles {
		if tl == stairs {
			g.p.x, g.p.y = i%g.lvl.w, i/g.lvl.w
		}
	}
	press(g, ">")
	require.Equal(t, 2, g.depth)
	require.Equal(t, g.lvl.startX, g.p.x)
}

func TestPlayRandomly(t *testing.T) {
	keys := []string{"h", "j", "k", "l", "y", "u", "b", "n", ".", ">"}
	for seed := range uint64(20) {
		g := newGame(seed)
		g.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		r := rand.New(rand.NewPCG(seed, 2))
		for range 3000 {
			if g.over {
				require.True(t, g.won || g.p.hp == 0)
				press(g, "r")
				require.False(t, g.over)
			}
			press(g, keys[r.IntN(len(keys))])
			if r.IntN(50) == 0 {
				// Cheat a little so the walk sees deeper levels.
				for i, tl := range g.lvl.tiles {
					if tl == stairs {
						g.p.x, g.p.y = i%g.lvl.w, i/g.lvl.w
					}
				}
			}
			g.Update(tickMsg{})
		}
		require.NotEmpty(t, g.render())
	}
}

func TestRenderSizes(t *testing.T) {
	g := newGame(3)
	for _, size := range [][2]int{{0, 0}, {20, 5}, {40, 12}, {80, 24}, {300, 100}} {
		g.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		out := g.render()
		if size[0] >= 40 {
			require.Len(t, strings.Split(out, "\n"), size[1])
		}
	}
	press(g, "?")
	require.Contains(t, g.render(), "Golden Brace")
	press(g, "x")
	require.False(t, g.showHelp)
}
