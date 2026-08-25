package importer

import (
	"math"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

type Polygon struct {
	Corners []rig.Vec3
	UVs     [][2]float64
	Texture int
}

type UVOrigin bool

const (
	UVTopLeft    UVOrigin = false
	UVBottomLeft UVOrigin = true
)

const snapDistance = 0.05

func MapFaces(
	polygons []Polygon,
	axes [3]rig.Vec3,
	width, height float64,
	origin UVOrigin,
) map[string]rig.Face {
	if len(polygons) == 0 || width <= 0 || height <= 0 {
		return nil
	}

	faces := make(map[string]rig.Face, 6)

	for _, polygon := range polygons {
		if polygon.Texture < 0 || len(polygon.Corners) < 3 {
			continue
		}

		side := sideOf(faceNormal(polygon.Corners), axes)

		if side == "" {
			continue
		}

		rect, ok := pixelRect(polygon.UVs, width, height, origin)

		if !ok {
			continue
		}

		if existing, seen := faces[side]; seen {
			rect = [4]float64{
				math.Min(existing.UV[0], rect[0]),
				math.Min(existing.UV[1], rect[1]),
				math.Max(existing.UV[2], rect[2]),
				math.Max(existing.UV[3], rect[3]),
			}
		}

		faces[side] = rig.Face{UV: rect, Texture: polygon.Texture}
	}

	if len(faces) == 0 {
		return nil
	}

	for side, face := range faces {
		for index := range face.UV {
			face.UV[index] = snap(face.UV[index])
		}

		faces[side] = face
	}

	return faces
}

func faceNormal(corners []rig.Vec3) rig.Vec3 {
	return cross(sub(corners[1], corners[0]), sub(corners[2], corners[0]))
}

func sideOf(normal rig.Vec3, axes [3]rig.Vec3) string {
	names := [3][2]string{{"east", "west"}, {"up", "down"}, {"south", "north"}}

	best := -1
	bestScore := 0.0
	positive := true

	for axis := 0; axis < 3; axis++ {
		score := dot(normal, axes[axis])

		if math.Abs(score) <= bestScore {
			continue
		}

		bestScore = math.Abs(score)
		best = axis
		positive = score > 0
	}

	if best < 0 {
		return ""
	}

	if positive {
		return names[best][0]
	}

	return names[best][1]
}

func pixelRect(uvs [][2]float64, width, height float64, origin UVOrigin) ([4]float64, bool) {
	rect := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	found := false

	for _, uv := range uvs {
		x := uv[0] * width
		y := uv[1] * height

		if origin == UVBottomLeft {
			y = (1 - uv[1]) * height
		}

		rect[0] = math.Min(rect[0], x)
		rect[1] = math.Min(rect[1], y)
		rect[2] = math.Max(rect[2], x)
		rect[3] = math.Max(rect[3], y)
		found = true
	}

	if !found {
		return [4]float64{}, false
	}

	return rect, true
}

func snap(value float64) float64 {
	nearest := math.Round(value)

	if math.Abs(value-nearest) <= snapDistance {
		return nearest
	}

	return math.Round(value*100) / 100
}
