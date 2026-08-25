package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/ai"
	"github.com/AlesixDev/MCAIA/backend/internal/animation"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter"
	"github.com/AlesixDev/MCAIA/backend/internal/importer"
	"github.com/AlesixDev/MCAIA/backend/internal/pipeline"
	"github.com/AlesixDev/MCAIA/backend/internal/store"
)

type projectSummary struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Format     string   `json:"format"`
	Notes      []string `json:"notes,omitempty"`
	Bones      int      `json:"bones"`
	Animations []string `json:"animations"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

func summarize(project *store.Project) projectSummary {
	names := make([]string, 0, len(project.Order))
	names = append(names, project.Order...)

	return projectSummary{
		ID:         project.ID,
		Name:       project.Name,
		Format:     project.Format,
		Notes:      project.Notes,
		Bones:      len(project.Rig.Bones),
		Animations: names,
		CreatedAt:  project.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  project.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func ownerID(r *http.Request) string {
	if user := currentUser(r); user != nil {
		return user.ID
	}

	return ""
}

const multipartMemory = 8 << 20

func (s *Server) owned(w http.ResponseWriter, r *http.Request) *store.Project {
	project, err := s.store.Get(r.PathValue("id"))

	if err != nil || !project.OwnedBy(ownerID(r)) {
		writeError(w, http.StatusNotFound, "project not found", nil)

		return nil
	}

	return project
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ready"
	detail := ""

	if err := s.engine.Available(r.Context()); err != nil {
		status = "engine_unavailable"
		detail = err.Error()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": status,
		"engine": s.engine.ID(),
		"detail": detail,
	})
}

func (s *Server) handleFormats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"formats":   exporter.List(),
		"importers": importer.List(),
	})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	request, err := readUpload(w, r, s.config.MaxUploadBytes)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "could not read the upload", err.Error())

		return
	}

	data := request.Data

	result, err := importer.Import(request)
	if err != nil {
		status := http.StatusBadRequest

		if errors.Is(err, importer.ErrUnknownFormat) {
			status = http.StatusUnsupportedMediaType
		}

		writeError(w, status, "could not read the model", err.Error())

		return
	}

	project, err := s.store.Create(ownerID(r), result.Name, result.Format, result.Rig, result.Notes, data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save the model", err.Error())

		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"project": summarize(project),
		"rig":     project.Rig,
	})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.List(ownerID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list the models", err.Error())

		return
	}

	items := make([]projectSummary, 0, len(projects))

	for _, project := range projects {
		items = append(items, summarize(project))
	}

	writeJSON(w, http.StatusOK, map[string]any{"projects": items})
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	project := s.owned(w, r)

	if project == nil {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"project":    summarize(project),
		"rig":        project.Rig,
		"animations": project.Ordered(),
	})
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	if s.owned(w, r) == nil {
		return
	}

	if err := s.store.Delete(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, "project not found", nil)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func readUpload(w http.ResponseWriter, r *http.Request, limit int64) (importer.Request, error) {
	body := http.MaxBytesReader(w, r.Body, limit)

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		data, err := io.ReadAll(body)
		if err != nil {
			return importer.Request{}, err
		}

		return importer.Request{Data: data, Filename: r.Header.Get("X-Mcaia-Filename")}, nil
	}

	r.Body = body

	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		return importer.Request{}, err
	}

	defer r.MultipartForm.RemoveAll()

	request := importer.Request{Assets: make(map[string][]byte)}

	for _, header := range r.MultipartForm.File["files"] {
		file, err := header.Open()
		if err != nil {
			return importer.Request{}, err
		}

		data, err := io.ReadAll(file)
		file.Close()

		if err != nil {
			return importer.Request{}, err
		}

		name := importer.BaseFile(header.Filename)
		request.Assets[strings.ToLower(name)] = data

		if request.Data != nil {
			continue
		}

		if _, err := importer.Detect(data, name); err == nil {
			request.Data = data
			request.Filename = name
		}
	}

	if request.Data == nil {
		return importer.Request{}, importer.ErrUnknownFormat
	}

	return request, nil
}

type renameBody struct {
	Name string `json:"name"`
}

func (s *Server) handleRenameProject(w http.ResponseWriter, r *http.Request) {
	project := s.owned(w, r)

	if project == nil {
		return
	}

	var body renameBody

	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())

		return
	}

	name := strings.TrimSpace(body.Name)

	if name == "" || len([]rune(name)) > 64 {
		writeError(w, http.StatusUnprocessableEntity, "name must be between 1 and 64 characters", nil)

		return
	}

	if err := s.store.Rename(project.ID, name); err != nil {
		writeError(w, http.StatusNotFound, "project not found", nil)

		return
	}

	project.Name = name

	writeJSON(w, http.StatusOK, map[string]any{"project": summarize(project)})
}

type generateBody struct {
	Prompt   string  `json:"prompt"`
	Name     string  `json:"name"`
	Duration float64 `json:"duration"`
	Loop     string  `json:"loop"`
	Style    string  `json:"style"`
	FPS      int     `json:"fps"`
	Optimize bool    `json:"optimize"`
}

func (s *Server) handleGenerateAnimation(w http.ResponseWriter, r *http.Request) {
	if s.owned(w, r) == nil {
		return
	}

	var body generateBody

	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())

		return
	}

	if body.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required", nil)

		return
	}

	ctx, cancel := contextWithTimeout(r, s.config.RequestTimeout)
	defer cancel()

	output, err := s.pipeline.Generate(ctx, pipeline.GenerateInput{
		ProjectID: r.PathValue("id"),
		Prompt:    body.Prompt,
		Name:      body.Name,
		Duration:  body.Duration,
		Loop:      animation.LoopMode(body.Loop),
		Style:     body.Style,
		FPS:       body.FPS,
		Optimize:  body.Optimize,
	})

	if err != nil {
		switch {
		case errors.Is(err, store.ErrProjectNotFound):
			writeError(w, http.StatusNotFound, "project not found", nil)
		case errors.Is(err, ai.ErrEngineUnavailable):
			writeError(w, http.StatusServiceUnavailable, "local ai engine unavailable", err.Error())
		default:
			writeError(w, http.StatusUnprocessableEntity, "generation failed", err.Error())
		}

		return
	}

	writeJSON(w, http.StatusOK, output)
}

func (s *Server) handleSaveAnimation(w http.ResponseWriter, r *http.Request) {
	if s.owned(w, r) == nil {
		return
	}

	var item animation.Animation

	if err := decodeJSON(r, &item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid animation payload", err.Error())

		return
	}

	warnings, err := s.pipeline.Save(r.PathValue("id"), &item)
	if err != nil {
		if errors.Is(err, store.ErrProjectNotFound) {
			writeError(w, http.StatusNotFound, "project not found", nil)

			return
		}

		writeError(w, http.StatusUnprocessableEntity, "animation rejected", warnings)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"animation": item, "warnings": warnings})
}

func (s *Server) handleDeleteAnimation(w http.ResponseWriter, r *http.Request) {
	if s.owned(w, r) == nil {
		return
	}

	err := s.store.DeleteAnimation(r.PathValue("id"), r.PathValue("name"))

	switch {
	case errors.Is(err, store.ErrProjectNotFound):
		writeError(w, http.StatusNotFound, "project not found", nil)
	case errors.Is(err, store.ErrAnimationNotFound):
		writeError(w, http.StatusNotFound, "animation not found", nil)
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal error", nil)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

type exportBody struct {
	Format    string   `json:"format"`
	Namespace string   `json:"namespace"`
	Names     []string `json:"names"`
	Download  bool     `json:"download"`
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if s.owned(w, r) == nil {
		return
	}

	var body exportBody

	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())

		return
	}

	if body.Format == "" {
		body.Format = "bedrock"
	}

	result, err := s.pipeline.Export(pipeline.ExportInput{
		ProjectID: r.PathValue("id"),
		Format:    body.Format,
		Namespace: body.Namespace,
		Names:     body.Names,
	})

	if err != nil {
		switch {
		case errors.Is(err, store.ErrProjectNotFound):
			writeError(w, http.StatusNotFound, "project not found", nil)
		case errors.Is(err, exporter.ErrUnknownFormat):
			writeError(w, http.StatusBadRequest, "unknown export format", nil)
		default:
			writeError(w, http.StatusUnprocessableEntity, "export failed", err.Error())
		}

		return
	}

	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(result.Data)))

	if body.Download {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+result.Filename+"\"")
	}

	w.Header().Set("X-Mcaia-Filename", result.Filename)
	w.WriteHeader(http.StatusOK)
	w.Write(result.Data)
}
