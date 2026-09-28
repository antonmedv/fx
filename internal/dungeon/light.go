package dungeon

import "math"

type rgb struct{ r, g, b float64 }

func (c rgb) add(o rgb) rgb       { return rgb{c.r + o.r, c.g + o.g, c.b + o.b} }
func (c rgb) mul(o rgb) rgb       { return rgb{c.r * o.r, c.g * o.g, c.b * o.b} }
func (c rgb) scale(k float64) rgb { return rgb{c.r * k, c.g * k, c.b * k} }
func (c rgb) brightness() float64 { return max(c.r, c.g, c.b) }

type light struct {
	x, y    int
	color   rgb
	radius  int
	flicker float64 // amplitude of the intensity wobble
	speed   float64 // how fast it wobbles
	phase   float64
	lum     []float64 // falloff per cell, baked once: the map never changes
}

// intensity is the brightness multiplier at animation frame t.
func (li *light) intensity(t int) float64 {
	if li.flicker == 0 {
		return 1
	}
	ft := float64(t) * li.speed
	return 1 + li.flicker*(0.6*math.Sin(ft*0.9+li.phase)+0.4*math.Sin(ft*2.3+li.phase*1.7))
}

// tone maps summed light to display brightness: dim light is lifted and
// overlapping lights saturate instead of clipping.
func tone(c rgb) rgb {
	f := func(v float64) float64 { return 1 - math.Exp(-2*v) }
	return rgb{f(c.r), f(c.g), f(c.b)}
}

func falloff(d float64, radius int) float64 {
	f := 1 - d/(float64(radius)+0.5)
	if f <= 0 {
		return 0
	}
	return f * f
}

func (l *level) bake(li *light) {
	li.lum = make([]float64, l.w*l.h)
	l.fov(li.x, li.y, li.radius, func(x, y int, d float64) {
		li.lum[l.idx(x, y)] = falloff(d, li.radius)
	})
}

// fov casts rays from (ox, oy) to every cell on the border of a square of the
// given radius. Each ray marks the cells it passes until it leaves the circle
// or hits an opaque cell, which is marked too. visit is called once per cell
// with the distance to it.
func (l *level) fov(ox, oy, radius int, visit func(x, y int, d float64)) {
	if !l.in(ox, oy) {
		return
	}
	l.stamp++
	var open [][2]int // transparent cells seen, for the wall pass below
	limit := float64(radius) + 0.5
	mark := func(x, y int) bool {
		d := math.Hypot(float64(x-ox), float64(y-oy))
		if !l.in(x, y) || d > limit {
			return false
		}
		i := l.idx(x, y)
		if l.marks[i] != l.stamp {
			l.marks[i] = l.stamp
			visit(x, y, d)
			if l.tiles[i].transparent() {
				open = append(open, [2]int{x, y})
			}
		}
		return l.tiles[i].transparent()
	}
	ray := func(tx, ty int) {
		// Bresenham from the origin to the target.
		dx, dy := abs(tx-ox), -abs(ty-oy)
		sx, sy := sign(tx-ox), sign(ty-oy)
		e := dx + dy
		x, y := ox, oy
		for x != tx || y != ty {
			e2 := 2 * e
			if e2 >= dy {
				e += dy
				x += sx
			}
			if e2 <= dx {
				e += dx
				y += sy
			}
			if !mark(x, y) {
				return
			}
		}
	}

	mark(ox, oy)
	for i := -radius; i <= radius; i++ {
		ray(ox+i, oy-radius)
		ray(ox+i, oy+radius)
		ray(ox-radius, oy+i)
		ray(ox+radius, oy+i)
	}

	// Rays slip past walls at grazing angles and leave holes in them. A wall
	// behind a visible floor cell, looking away from the origin, is visible.
	// Straight in line with the origin, walls on both sides count.
	away := func(d int) []int {
		if d == 0 {
			return []int{-1, 0, 1}
		}
		return []int{0, d}
	}
	for _, c := range open {
		x, y := c[0], c[1]
		for _, dx := range away(sign(x - ox)) {
			for _, dy := range away(sign(y - oy)) {
				if !l.at(x+dx, y+dy).transparent() {
					mark(x+dx, y+dy)
				}
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	}
	return 0
}
