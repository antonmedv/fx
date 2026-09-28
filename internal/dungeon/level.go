package dungeon

import "math/rand/v2"

const (
	mapW     = 96
	mapH     = 40
	maxDepth = 8
)

type tile uint8

const (
	wall tile = iota
	floor
	stairs
	brazier
	crystal
)

func (t tile) passable() bool    { return t == floor || t == stairs }
func (t tile) transparent() bool { return t != wall }

type rect struct{ x, y, w, h int }

func (r rect) center() (int, int) { return r.x + r.w/2, r.y + r.h/2 }

// overlaps reports whether the rooms touch, keeping one wall between them.
func (r rect) overlaps(o rect) bool {
	return r.x-1 <= o.x+o.w && o.x-1 <= r.x+r.w && r.y-1 <= o.y+o.h && o.y-1 <= r.y+r.h
}

type level struct {
	w, h     int
	tiles    []tile
	seen     []bool
	rooms    []rect
	lights   []*light
	monsters []*monster
	items    []*item
	startX   int
	startY   int

	marks []int // fov dedup stamps
	stamp int
}

func newLevel(w, h int) *level {
	return &level{
		w:     w,
		h:     h,
		tiles: make([]tile, w*h),
		seen:  make([]bool, w*h),
		marks: make([]int, w*h),
	}
}

func (l *level) in(x, y int) bool     { return x >= 0 && y >= 0 && x < l.w && y < l.h }
func (l *level) idx(x, y int) int     { return y*l.w + x }
func (l *level) set(x, y int, t tile) { l.tiles[l.idx(x, y)] = t }

func (l *level) at(x, y int) tile {
	if !l.in(x, y) {
		return wall
	}
	return l.tiles[l.idx(x, y)]
}

func (l *level) monsterAt(x, y int) *monster {
	for _, m := range l.monsters {
		if m.x == x && m.y == y {
			return m
		}
	}
	return nil
}

func (l *level) itemAt(x, y int) *item {
	for _, it := range l.items {
		if it.x == x && it.y == y {
			return it
		}
	}
	return nil
}

var dirs8 = [8][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

// distances returns the number of steps from (x, y) to every cell, or -1 for
// cells that cannot be reached.
func (l *level) distances(x, y int) []int {
	dist := make([]int, l.w*l.h)
	for i := range dist {
		dist[i] = -1
	}
	dist[l.idx(x, y)] = 0
	queue := []int{l.idx(x, y)}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		cx, cy := i%l.w, i/l.w
		for _, d := range dirs8 {
			nx, ny := cx+d[0], cy+d[1]
			if !l.at(nx, ny).passable() {
				continue
			}
			j := l.idx(nx, ny)
			if dist[j] < 0 {
				dist[j] = dist[i] + 1
				queue = append(queue, j)
			}
		}
	}
	return dist
}

func (l *level) reachable(x, y int) int {
	n := 0
	for _, d := range l.distances(x, y) {
		if d >= 0 {
			n++
		}
	}
	return n
}

func (l *level) passableCount() int {
	n := 0
	for _, t := range l.tiles {
		if t.passable() {
			n++
		}
	}
	return n
}

func generate(r *rand.Rand, depth int) *level {
	l := newLevel(mapW, mapH)

	for tries := 0; tries < 300 && len(l.rooms) < 14; tries++ {
		w, h := 5+r.IntN(10), 4+r.IntN(5)
		room := rect{1 + r.IntN(l.w-w-1), 1 + r.IntN(l.h-h-1), w, h}
		ok := true
		for _, o := range l.rooms {
			if room.overlaps(o) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		for y := room.y; y < room.y+room.h; y++ {
			for x := room.x; x < room.x+room.w; x++ {
				l.set(x, y, floor)
			}
		}
		if len(l.rooms) > 0 {
			l.corridor(r, l.rooms[len(l.rooms)-1], room)
		}
		l.rooms = append(l.rooms, room)
	}
	for i := 0; i < 3; i++ { // a few loops
		l.corridor(r, l.rooms[r.IntN(len(l.rooms))], l.rooms[r.IntN(len(l.rooms))])
	}

	l.startX, l.startY = l.rooms[0].center()

	// The exit goes into the room farthest from the start.
	dist := l.distances(l.startX, l.startY)
	last := 0
	for i, room := range l.rooms {
		x, y := room.center()
		lx, ly := l.rooms[last].center()
		if dist[l.idx(x, y)] > dist[l.idx(lx, ly)] {
			last = i
		}
	}
	exitX, exitY := l.rooms[last].center()
	if depth < maxDepth {
		l.set(exitX, exitY, stairs)
	}

	l.decorate(r)

	if depth == maxDepth {
		l.items = append(l.items, &item{kind: brace, x: exitX, y: exitY})
		l.lights = append(l.lights, &light{x: exitX, y: exitY, color: rgb{1, 0.85, 0.3}, radius: 6, flicker: 0.1, speed: 0.5})
	}

	l.populate(r, depth, exitX, exitY)

	for _, li := range l.lights {
		l.bake(li)
	}
	return l
}

func (l *level) corridor(r *rand.Rand, a, b rect) {
	ax, ay := a.center()
	bx, by := b.center()
	carve := func(x, y int) {
		if l.at(x, y) == wall {
			l.set(x, y, floor)
		}
	}
	h := func(x1, x2, y int) {
		for x := min(x1, x2); x <= max(x1, x2); x++ {
			carve(x, y)
		}
	}
	v := func(y1, y2, x int) {
		for y := min(y1, y2); y <= max(y1, y2); y++ {
			carve(x, y)
		}
	}
	if r.IntN(2) == 0 {
		h(ax, bx, ay)
		v(ay, by, bx)
	} else {
		v(ay, by, ax)
		h(ax, bx, by)
	}
}

// decorate puts light sources into the corners of some rooms.
func (l *level) decorate(r *rand.Rand) {
	for i, room := range l.rooms {
		var t tile
		var li *light
		switch p := r.Float64(); {
		case i == 0 || p < 0.45:
			t = brazier
			li = &light{color: rgb{1, 0.55, 0.2}, radius: 7, flicker: 0.18, speed: 1}
		case p < 0.65:
			t = crystal
			li = &light{color: rgb{0.35, 0.6, 1}, radius: 6, flicker: 0.12, speed: 0.2}
		default:
			continue
		}
		corners := [4][2]int{
			{room.x, room.y},
			{room.x + room.w - 1, room.y},
			{room.x, room.y + room.h - 1},
			{room.x + room.w - 1, room.y + room.h - 1},
		}
		start := r.IntN(4)
		for k := range 4 {
			c := corners[(start+k)%4]
			if l.at(c[0], c[1]) != floor {
				continue
			}
			before := l.reachable(l.startX, l.startY)
			l.set(c[0], c[1], t)
			if l.reachable(l.startX, l.startY) != before-1 {
				l.set(c[0], c[1], floor) // it would cut the map in two
				continue
			}
			li.x, li.y = c[0], c[1]
			li.phase = r.Float64() * 10
			l.lights = append(l.lights, li)
			break
		}
	}
}

// free returns a random empty floor cell in the room.
func (l *level) free(r *rand.Rand, room rect) (int, int, bool) {
	for range 20 {
		x, y := room.x+r.IntN(room.w), room.y+r.IntN(room.h)
		if l.at(x, y) == floor && l.monsterAt(x, y) == nil && l.itemAt(x, y) == nil &&
			(x != l.startX || y != l.startY) {
			return x, y, true
		}
	}
	return 0, 0, false
}

func (l *level) populate(r *rand.Rand, depth, exitX, exitY int) {
	var kinds []*kind
	for _, k := range bestiary {
		if k.minDepth <= depth && depth <= k.maxDepth {
			kinds = append(kinds, k)
		}
	}
	for range 4 + depth*2 {
		room := l.rooms[r.IntN(len(l.rooms))]
		if len(l.rooms) > 1 {
			room = l.rooms[1+r.IntN(len(l.rooms)-1)] // let the player wake up in peace
		}
		if x, y, ok := l.free(r, room); ok {
			k := kinds[r.IntN(len(kinds))]
			l.monsters = append(l.monsters, &monster{kind: k, x: x, y: y, hp: k.hp})
		}
	}
	if depth == maxDepth {
		// The dragon sleeps next to its treasure.
		for _, d := range dirs8 {
			x, y := exitX+d[0], exitY+d[1]
			if l.at(x, y) == floor && l.monsterAt(x, y) == nil {
				l.monsters = append(l.monsters, &monster{kind: dragon, x: x, y: y, hp: dragon.hp})
				break
			}
		}
	}

	drop := func(k itemKind, n int) {
		for range n {
			if x, y, ok := l.free(r, l.rooms[r.IntN(len(l.rooms))]); ok {
				l.items = append(l.items, &item{kind: k, x: x, y: y, amount: 1 + r.IntN(5+depth*5)})
			}
		}
	}
	drop(potion, 1+r.IntN(2))
	drop(gold, 3+r.IntN(3))
	drop(torch, 1+r.IntN(2))
	if r.Float64() < 0.4 {
		drop(weapon, 1)
	}
	if r.Float64() < 0.4 {
		drop(armor, 1)
	}
}
