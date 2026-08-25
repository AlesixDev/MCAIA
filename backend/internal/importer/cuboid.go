package importer

import (
	"math"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

const (
	cuboidCorners   = 8
	cuboidTolerance = 1e-3
)

type frame struct {
	axes [3]rig.Vec3
	size rig.Vec3
}

type Box struct {
	From     rig.Vec3
	To       rig.Vec3
	Origin   rig.Vec3
	Rotation rig.Vec3
	Axes     [3]rig.Vec3
}

var identity = [3]rig.Vec3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}

func Cuboid(points []rig.Vec3) (Box, bool) {
	corners := distinct(points)

	if len(corners) != cuboidCorners {
		return Box{Axes: identity}, false
	}

	edges, found := edgeVectors(corners)

	if !found {
		return Box{Axes: identity}, false
	}

	basis := orient(edges)
	centre := add(corners[0], scale(add(add(edges[0], edges[1]), edges[2]), 0.5))
	half := scale(basis.size, 0.5)

	return Box{
		From:     sub(centre, half),
		To:       add(centre, half),
		Origin:   centre,
		Rotation: eulerZYX(basis.axes),
		Axes:     basis.axes,
	}, true
}

func distinct(points []rig.Vec3) []rig.Vec3 {
	unique := make([]rig.Vec3, 0, cuboidCorners)

	for _, point := range points {
		duplicate := false

		for _, kept := range unique {
			if length(sub(point, kept)) <= cuboidTolerance {
				duplicate = true

				break
			}
		}

		if duplicate {
			continue
		}

		if len(unique) == cuboidCorners {
			return append(unique, point)
		}

		unique = append(unique, point)
	}

	return unique
}

func edgeVectors(corners []rig.Vec3) ([3]rig.Vec3, bool) {
	var edges [3]rig.Vec3

	origin := corners[0]
	deltas := make([]rig.Vec3, 0, cuboidCorners-1)

	for _, corner := range corners[1:] {
		deltas = append(deltas, sub(corner, origin))
	}

	for first := 0; first < len(deltas); first++ {
		for second := first + 1; second < len(deltas); second++ {
			for third := second + 1; third < len(deltas); third++ {
				candidate := [3]rig.Vec3{deltas[first], deltas[second], deltas[third]}

				if !perpendicular(candidate) {
					continue
				}

				if !predicts(origin, candidate, corners) {
					continue
				}

				return candidate, true
			}
		}
	}

	return edges, false
}

func perpendicular(edges [3]rig.Vec3) bool {
	for index := 0; index < 3; index++ {
		next := edges[(index+1)%3]
		current := edges[index]

		scaleOf := length(current) * length(next)

		if scaleOf == 0 || math.Abs(dot(current, next))/scaleOf > cuboidTolerance {
			return false
		}
	}

	return true
}

func predicts(origin rig.Vec3, edges [3]rig.Vec3, corners []rig.Vec3) bool {
	wanted := []rig.Vec3{
		add(origin, add(edges[0], edges[1])),
		add(origin, add(edges[0], edges[2])),
		add(origin, add(edges[1], edges[2])),
		add(origin, add(edges[0], add(edges[1], edges[2]))),
	}

	for _, point := range wanted {
		if !contains(corners, point) {
			return false
		}
	}

	return true
}

func contains(corners []rig.Vec3, point rig.Vec3) bool {
	for _, corner := range corners {
		if length(sub(corner, point)) <= cuboidTolerance {
			return true
		}
	}

	return false
}

func orient(edges [3]rig.Vec3) frame {
	var result frame

	used := [3]bool{}

	for axis := 0; axis < 3; axis++ {
		best := -1
		bestScore := -1.0

		for index := 0; index < 3; index++ {
			if used[index] {
				continue
			}

			size := length(edges[index])

			if size == 0 {
				continue
			}

			score := math.Abs(edges[index][axis]) / size

			if score > bestScore {
				bestScore = score
				best = index
			}
		}

		if best < 0 {
			return frame{axes: identity, size: rig.Vec3{}}
		}

		used[best] = true

		size := length(edges[best])
		direction := scale(edges[best], 1/size)

		if direction[axis] < 0 {
			direction = scale(direction, -1)
		}

		result.axes[axis] = direction
		result.size[axis] = size
	}

	if dot(cross(result.axes[0], result.axes[1]), result.axes[2]) < 0 {
		result.axes[2] = scale(result.axes[2], -1)
	}

	return result
}

func eulerZYX(axes [3]rig.Vec3) rig.Vec3 {
	m11, m21, m31 := axes[0][0], axes[0][1], axes[0][2]
	m12, m22, m32 := axes[1][0], axes[1][1], axes[1][2]
	_, _, m33 := axes[2][0], axes[2][1], axes[2][2]

	y := math.Asin(-clampUnit(m31))

	var x, z float64

	if math.Abs(m31) < 0.9999999 {
		x = math.Atan2(m32, m33)
		z = math.Atan2(m21, m11)
	} else {
		x = 0
		z = math.Atan2(-m12, m22)
	}

	return rig.Vec3{toDegrees(x), toDegrees(y), toDegrees(z)}
}

func clampUnit(value float64) float64 {
	return math.Max(-1, math.Min(1, value))
}

func toDegrees(radians float64) float64 {
	return math.Round(radians*180/math.Pi*1000) / 1000
}

func add(a, b rig.Vec3) rig.Vec3 {
	return rig.Vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func sub(a, b rig.Vec3) rig.Vec3 {
	return rig.Vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func scale(a rig.Vec3, factor float64) rig.Vec3 {
	return rig.Vec3{a[0] * factor, a[1] * factor, a[2] * factor}
}

func dot(a, b rig.Vec3) float64 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

func cross(a, b rig.Vec3) rig.Vec3 {
	return rig.Vec3{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func length(a rig.Vec3) float64 {
	return math.Sqrt(dot(a, a))
}
