<div align="center">
  <h1>mcaia</h1>
  <p><b>Animate Minecraft models by describing the motion, with a model that runs on your own machine.</b></p>
  <p>Drop in a Blockbench project, a glTF or an OBJ, say what you want it to do, and get keyframes back — validated against the rig, ready to open in Blockbench.</p>
  <sub>Go · Next.js · Ollama · nothing leaves your machine</sub>
  <p>
    <a href="https://github.com/AlesixDev/MCAIA/actions/workflows/ci.yml"><img src="https://github.com/AlesixDev/MCAIA/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT" /></a>
  </p>
</div>

## Features

**Import**
- Blockbench `.bbmodel` with its hierarchy, pivots, box UV and embedded textures
- glTF and GLB read for real: mesh geometry, node hierarchy and the embedded image
- Wavefront OBJ, with the `.mtl` and its texture when you upload them alongside
- Block units converted to Blockbench pixels, and rotated cubes recovered as oriented boxes rather than inflated bounding boxes

**Generation**
- The model never writes raw keyframes: it describes the motion as a list of moves, and the engine turns each one into curves
- Moves take a time window, so a dance is a sequence of beats instead of one long oscillation
- Follow-through, anticipation and overshoot added by the engine
- Every plan is repaired and validated against the rig before it becomes an animation: unknown bones are dropped, directions corrected, loops closed

**Rig safety**
- Translations move the whole character, never one loose cube
- Skin overlays stay welded to the limb they wrap
- Per-part limits applied to the combined result, not just to each move

**Export**
- `.bbmodel` with the animations embedded and the animators bound to their bones
- Bedrock `.animation.json` and GeckoLib
- Non-animatable source formats promoted so Blockbench actually shows the Animate tab

## Getting started

Requires **Go 1.26+**, **Node 20+** and an [Ollama](https://ollama.com) instance with a model pulled.

```sh
git clone https://github.com/AlesixDev/MCAIA
cd mcaia
ollama pull qwen3:14b
```

```sh
cd backend && make run
```

```sh
cd frontend && npm install && npm run dev
```

The API listens on `127.0.0.1:8787` and the interface on `localhost:3000`.

### Build

```sh
cd backend && make build      # single binary for the current platform
cd backend && make dist       # linux/amd64, linux/arm64 and windows/amd64 into dist/
cd frontend && npm run build
```

### Make targets

| Target | What it does |
|---|---|
| `build` | Build `mcaia` for the current platform |
| `run` | Run the API from source |
| `dist` | Cross-compile linux/amd64, linux/arm64 and windows/amd64 |
| `test` | Everything |
| `test-unit` | `internal/...` — fast, no database or sockets |
| `test-integration` | `test/...` — import, generate and export end to end |
| `vet` | `go vet ./...` |
| `fmt` | `go fmt ./...` |
| `clean` | Remove the binary and `dist/` |

## Configuration

Every key is an environment variable read at startup, with a working default.

| Variable | Default | Holds |
|---|---|---|
| `MCAIA_ADDR` | `127.0.0.1:8787` | Where the API listens |
| `MCAIA_ORIGIN` | `http://localhost:3000` | Allowed CORS origin |
| `MCAIA_OLLAMA_URL` | `http://127.0.0.1:11434` | Ollama instance |
| `MCAIA_OLLAMA_MODEL` | `qwen3:14b` | Model used to plan the motion |
| `MCAIA_TEMPERATURE` | `0.4` | Sampling temperature |
| `MCAIA_NUM_CTX` | `8192` | Context window |
| `MCAIA_TIMEOUT_SECONDS` | `180` | Generation timeout |
| `MCAIA_MAX_UPLOAD_MB` | `32` | Upload ceiling |
| `MCAIA_DATA_DIR` | `data` | SQLite database and avatars |

The frontend reads `NEXT_PUBLIC_API_URL`, defaulting to `http://127.0.0.1:8787`.

## How a prompt becomes keyframes

The model is never asked for numbers on a timeline. It is asked for a **plan**: a list of moves, each one a target, a motion type, an axis, an amplitude, a number of cycles, and the window of the clip it occupies.

```json
{"target": "arms", "motion": "swing", "axis": "z", "amplitude": 40,
 "cycles": 2, "alternate": true, "start": 0.55, "end": 1}
```

That plan then goes through four stages before it is an animation:

| Stage | Package | What happens |
|---|---|---|
| Repair | `motion` | Aliases normalised, duplicates dropped, directions corrected against the prompt, cycles rounded so a loop closes |
| Synthesize | `motion` | Each move becomes a curve over its window, blended in and out; follow-through added to real joints |
| Validate | `animation` | Bones checked against the rig, values clamped, unknown targets reported as warnings |
| Polish | `animation` | Snapped to the frame rate, redundant keyframes removed |

Windows are what make a routine possible. Without them every move ran the whole clip, so any two dances came out as the same oscillation with different numbers.

## Architecture

```
.
├── backend/
│   ├── cmd/mcaia/           # Entry point: wires importers, exporters and the API
│   ├── internal/
│   │   ├── ai/              # Prompt, response schema, Ollama client
│   │   ├── animation/       # Keyframe types, validation, polish, optimisation
│   │   ├── auth/            # Accounts and sessions
│   │   ├── bbmodel/         # Blockbench project parsing
│   │   ├── config/          # Environment configuration
│   │   ├── database/        # SQLite
│   │   ├── exporter/        # bbmodel, Bedrock and GeckoLib writers
│   │   ├── httpapi/         # Routes, middleware, uploads
│   │   ├── importer/        # bbmodel, glTF and OBJ readers
│   │   ├── motion/          # The plan: roles, repair, synthesis, presets
│   │   ├── pipeline/        # Generate and export, end to end
│   │   ├── rig/             # Skeleton, bones, cubes, textures
│   │   └── store/           # Projects and animations
│   └── test/integration/    # Import, generate and export against a real database
└── frontend/
    ├── app/                 # Routes: welcome, project, inspector, library, login
    ├── components/          # Shell, chat, 3D preview, primitives
    └── lib/                 # The only layer that talks to the API
```

| Package | Responsibility |
|---|---|
| `importer` | Turning three very different file formats into one rig, with the same geometry and UVs |
| `rig` | The skeleton every other package agrees on |
| `motion` | Everything between a plan and a set of curves: roles, limits, windows, follow-through |
| `animation` | The keyframes themselves, and whether they are valid for this rig |
| `pipeline` | The only place that knows the whole sequence |
| `exporter` | Writing to formats Blockbench and GeckoLib will open |

### Routes

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/health` | Whether the engine is reachable |
| `GET` | `/api/v1/formats` | Importers and exporters available |
| `POST` | `/api/v1/projects` | Upload a model, raw or multipart with its companions |
| `GET` | `/api/v1/projects` | List models |
| `GET` | `/api/v1/projects/{id}` | Project, rig and animations |
| `PATCH` | `/api/v1/projects/{id}` | Rename |
| `DELETE` | `/api/v1/projects/{id}` | Delete |
| `POST` | `/api/v1/projects/{id}/animations/generate` | Describe a motion, get an animation |
| `POST` | `/api/v1/projects/{id}/animations` | Save an animation directly |
| `DELETE` | `/api/v1/projects/{id}/animations/{name}` | Delete one |
| `POST` | `/api/v1/projects/{id}/export` | Export to a format |

Registration, login, logout, profile and avatars live under `/api/v1/auth`, `/api/v1/me` and `/api/v1/avatars`.

## Getting a model in

**`.bbmodel` is the format that loses nothing.** Geometry, hierarchy, pivots, box UV and the texture all travel inside a single file, and the export path reuses your original document rather than rebuilding it.

**glTF and GLB are the next best thing.** A Blockbench glTF export is self-contained: the geometry is in accessors, the image is a data URI, and the node tree carries real joints. Everything is read.

**OBJ loses the most, by design.** It stores no hierarchy and no pivots, so every group arrives as a loose bone pivoted at the centre of its own geometry — which means animating it pulls the model apart. Textures need the `.mtl` and the image uploaded alongside the `.obj`. Prefer either of the other two.

## Limitations

**Quality is bounded by the model you run.** The engine guarantees the animation is structurally sound: nothing detaches, limits hold, loops close. Whether the choreography is any *good* comes from the model choosing amplitudes and windows, and a 14B running locally has a ceiling.

**Repeated prompts can still converge.** Windows make different shapes possible, but nothing yet tells the model what it already generated for this project, so asking twice for a dance can land somewhere similar.

**OBJ rigs animate badly and there is no way around it in the format.** Inferring hierarchy and joints from part names is possible and not implemented.

**Face UVs are taken as an axis-aligned rectangle.** Correct for box-shaped models, which is everything here; a flipped or rotated face mapping is not preserved.

## Development

```sh
cd backend
make test-unit          # fast, no database or sockets
make test-integration   # import, generate and export end to end
make test               # both
make vet
```

Tests are split by what they need:

- **Unit tests live next to their package** (`internal/*/`) because they reach into unexported internals — the oriented-box fit, time windows, per-part limits, the repair pass. Go cannot test unexported identifiers from another directory, so these cannot move.
- **Integration tests live in `test/integration/`**. They open a real SQLite database, import a model, run the pipeline with a stub engine and export the result, so they only use the public API and take seconds rather than milliseconds.

Every push and pull request runs the same thing in CI: `gofmt`, `go vet`, `go build`, the unit tests and the integration tests for the backend, and a typecheck plus a production build for the frontend.

`internal/importer` covers the oriented-box recovery against known rotations, including an elongated box where picking the three shortest edges gives the wrong answer. `internal/motion` covers windows, the blend at section edges, overlay pruning, follow-through, combined limits, and that a body translation moves every root so the model stays in one piece. `test/integration` covers the whole path from a Blockbench file to a Bedrock timeline, including that generating twice does not overwrite the first animation.

## Tech stack

| Layer | Technology |
|---|---|
| API | Go 1.26, `net/http` routing, SQLite |
| Model | Ollama, structured output against a JSON schema |
| Interface | Next.js 16 App Router, React 19, Tailwind v4 |
| Preview | three.js, matching Blockbench's axis and sign conventions |
| Formats | Blockbench, Bedrock, GeckoLib, glTF, OBJ |

## License

MIT — see [LICENSE](./LICENSE).
