package importer

import (
	"math"
	"testing"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

func corners(centre rig.Vec3, size rig.Vec3, rotation rig.Vec3) []rig.Vec3 {
	radians := func(value float64) float64 { return value * math.Pi / 180 }

	sx, cx := math.Sincos(radians(rotation[0]))
	sy, cy := math.Sincos(radians(rotation[1]))
	sz, cz := math.Sincos(radians(rotation[2]))

	matrix := [3][3]float64{
		{cz * cy, cz*sy*sx - sz*cx, cz*sy*cx + sz*sx},
		{sz * cy, sz*sy*sx + cz*cx, sz*sy*cx - cz*sx},
		{-sy, cy * sx, cy * cx},
	}

	points := make([]rig.Vec3, 0, 8)

	for _, x := range []float64{-size[0] / 2, size[0] / 2} {
		for _, y := range []float64{-size[1] / 2, size[1] / 2} {
			for _, z := range []float64{-size[2] / 2, size[2] / 2} {
				local := rig.Vec3{x, y, z}
				point := rig.Vec3{}

				for row := 0; row < 3; row++ {
					point[row] = matrix[row][0]*local[0] + matrix[row][1]*local[1] +
						matrix[row][2]*local[2] + centre[row]
				}

				points = append(points, point)
			}
		}
	}

	return points
}

func close(t *testing.T, label string, got, want rig.Vec3, tolerance float64) {
	t.Helper()

	for axis := 0; axis < 3; axis++ {
		if math.Abs(got[axis]-want[axis]) > tolerance {
			t.Errorf("%s = %v, want %v", label, got, want)

			return
		}
	}
}

func TestCuboidRecoversRotatedBox(t *testing.T) {
	cases := []struct {
		name     string
		centre   rig.Vec3
		size     rig.Vec3
		rotation rig.Vec3
	}{
		{"axis aligned", rig.Vec3{0, 18, 0}, rig.Vec3{8, 12, 4}, rig.Vec3{0, 0, 0}},
		{"tilted head", rig.Vec3{0, 28, 0}, rig.Vec3{8, 8, 8}, rig.Vec3{6, -5, 0}},
		{"elongated limb", rig.Vec3{-2, 6, 0}, rig.Vec3{4, 12, 4}, rig.Vec3{0, 12, 0}},
		{"three axes", rig.Vec3{1, 2, 3}, rig.Vec3{6, 10, 2}, rig.Vec3{10, 20, 30}},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			box, ok := Cuboid(corners(item.centre, item.size, item.rotation))

			if !ok {
				t.Fatal("Cuboid did not recognise the box")
			}

			close(t, "origin", box.Origin, item.centre, 1e-6)
			close(t, "size", sub(box.To, box.From), item.size, 1e-6)
			close(t, "rotation", box.Rotation, item.rotation, 1e-3)
		})
	}
}

func TestCuboidRejectsNonBoxes(t *testing.T) {
	pyramid := []rig.Vec3{
		{0, 0, 0}, {4, 0, 0}, {4, 0, 4}, {0, 0, 4}, {2, 5, 2},
	}

	if _, ok := Cuboid(pyramid); ok {
		t.Error("Cuboid accepted a shape that is not a box")
	}
}
