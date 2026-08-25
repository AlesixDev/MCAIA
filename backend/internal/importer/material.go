package importer

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/AlesixDev/MCAIA/backend/internal/rig"
)

type materialLibrary struct {
	textures []rig.Texture
	byName   map[string]int
}

func (m *materialLibrary) index(material string) int {
	if m == nil {
		return -1
	}

	found, ok := m.byName[material]

	if !ok {
		return -1
	}

	return found
}

func readMaterials(libraries []string, request Request) (*materialLibrary, []string) {
	result := &materialLibrary{byName: make(map[string]int)}
	problems := make([]string, 0)
	sources := make(map[string]int)

	for _, library := range libraries {
		data, ok := request.Asset(library)

		if !ok {
			problems = append(problems, fmt.Sprintf("%s was not uploaded, so the model has no textures.", BaseFile(library)))

			continue
		}

		for material, file := range parseMTL(data) {
			image, ok := request.Asset(file)

			if !ok {
				problems = append(problems, fmt.Sprintf("%s references %s, which was not uploaded.", BaseFile(library), BaseFile(file)))

				continue
			}

			key := strings.ToLower(BaseFile(file))

			if existing, seen := sources[key]; seen {
				result.byName[material] = existing

				continue
			}

			texture, err := decodeTexture(BaseFile(file), image)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s could not be read: %s", BaseFile(file), err))

				continue
			}

			result.textures = append(result.textures, texture)
			sources[key] = len(result.textures) - 1
			result.byName[material] = len(result.textures) - 1
		}
	}

	return result, problems
}

func parseMTL(data []byte) map[string]string {
	materials := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	current := ""

	for scanner.Scan() {
		fields := strings.Fields(strings.TrimSpace(scanner.Text()))

		if len(fields) < 2 {
			continue
		}

		switch strings.ToLower(fields[0]) {
		case "newmtl":
			current = fields[1]

		case "map_kd", "map_ka":
			if current == "" {
				continue
			}

			if _, taken := materials[current]; taken {
				continue
			}

			materials[current] = mapPath(fields[1:])
		}
	}

	return materials
}

func mapPath(fields []string) string {
	for index := 0; index < len(fields); index++ {
		if strings.HasPrefix(fields[index], "-") {
			continue
		}

		return strings.Join(fields[index:], " ")
	}

	return ""
}

func decodeTexture(name string, data []byte) (rig.Texture, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return rig.Texture{}, err
	}

	mime := "image/png"

	if format == "jpeg" {
		mime = "image/jpeg"
	}

	return rig.Texture{
		Name:   name,
		Source: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
		Width:  config.Width,
		Height: config.Height,
	}, nil
}
