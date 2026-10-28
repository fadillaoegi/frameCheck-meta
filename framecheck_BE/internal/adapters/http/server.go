// Package http maps HTTP requests to application use cases.
package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"framecheck/internal/application"
	"framecheck/internal/domain"
)

const (
	maxJSONBody = 64 << 10
	// multipart overhead on top of the image limit
	maxUploadBody = domain.MaxImageBytes + 64<<10
)

type Server struct {
	svc *application.Service
	log *slog.Logger
}

func NewHandler(svc *application.Service, log *slog.Logger) http.Handler {
	s := &Server{svc: svc, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /v1/projects", s.listProjects)
	mux.HandleFunc("POST /v1/projects", s.createProject)
	mux.HandleFunc("GET /v1/projects/{id}", s.getProject)
	mux.HandleFunc("POST /v1/projects/{id}/assets", s.uploadAsset)
	mux.HandleFunc("GET /v1/assets/{id}/content", s.assetContent)
	mux.HandleFunc("POST /v1/projects/{id}/comparisons", s.createComparison)
	mux.HandleFunc("GET /v1/comparisons/{id}", s.getComparison)
	mux.HandleFunc("POST /v1/comparisons/{id}/runs", s.startRun)
	mux.HandleFunc("GET /v1/runs/{id}", s.getRun)
	mux.HandleFunc("GET /v1/runs/{id}/export", s.exportRun)
	mux.HandleFunc("POST /v1/findings/{id}/reviews", s.reviewFinding)
	return s.middleware(mux)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		// Path only: no query strings, bodies, or image bytes in logs.
		s.log.Info("http", "method", r.Method, "path", r.URL.Path, "status", rec.status,
			"ms", time.Since(start).Milliseconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	var b errorBody
	b.Error.Code, b.Error.Message = code, msg
	writeJSON(w, status, b)
}

// fail maps domain errors to HTTP and hides internal errors behind a generic
// message so paths, SQL, or provider details never reach the client.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	if de, ok := domain.AsError(err); ok {
		status := map[domain.ErrorKind]int{
			domain.KindInvalid:     http.StatusBadRequest,
			domain.KindNotFound:    http.StatusNotFound,
			domain.KindConflict:    http.StatusConflict,
			domain.KindTooLarge:    http.StatusRequestEntityTooLarge,
			domain.KindUnavailable: http.StatusServiceUnavailable,
		}[de.Kind]
		if status == 0 {
			status = http.StatusInternalServerError
		}
		writeError(w, status, de.Code, de.Message)
		return
	}
	s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "INTERNAL", "unexpected server error")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.Invalid("INVALID_JSON", "request body must be valid JSON for this endpoint")
	}
	if dec.More() {
		return domain.Invalid("INVALID_JSON", "request body must contain one JSON object")
	}
	return nil
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	ps, err := s.svc.ListProjects(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]projectDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, toProject(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": out})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	p, err := s.svc.CreateProject(r.Context(), body.Name)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProject(p))
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	pd, err := s.svc.GetProject(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := projectDetailDTO{projectDTO: toProject(pd.Project),
		Assets: make([]assetDTO, 0, len(pd.Assets)), Comparisons: make([]comparisonDTO, 0, len(pd.Comparisons))}
	for _, a := range pd.Assets {
		out.Assets = append(out.Assets, toAsset(a))
	}
	for _, c := range pd.Comparisons {
		out.Comparisons = append(out.Comparisons, toComparison(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) uploadAsset(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBody)
	mr, err := r.MultipartReader()
	if err != nil {
		s.fail(w, r, domain.Invalid("INVALID_UPLOAD", "request must be multipart/form-data"))
		return
	}

	fields := map[string]string{}
	var data []byte
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.fail(w, r, uploadReadError(err))
			return
		}
		name := part.FormName()
		if name == "file" {
			if data != nil {
				s.fail(w, r, domain.Invalid("INVALID_UPLOAD", "send exactly one file"))
				return
			}
			data, err = io.ReadAll(io.LimitReader(part, domain.MaxImageBytes+1))
			if err != nil {
				s.fail(w, r, uploadReadError(err))
				return
			}
			if len(data) > domain.MaxImageBytes {
				s.fail(w, r, domain.TooLarge("IMAGE_TOO_LARGE", "image must be at most %d MB", domain.MaxImageBytes>>20))
				return
			}
			continue
		}
		v, err := io.ReadAll(io.LimitReader(part, 257))
		if err != nil || len(v) > 256 {
			s.fail(w, r, domain.Invalid("INVALID_UPLOAD", "form field %q is too long", name))
			return
		}
		fields[name] = strings.TrimSpace(string(v))
	}
	if data == nil {
		s.fail(w, r, domain.Invalid("INVALID_UPLOAD", "file is required"))
		return
	}

	vw, err1 := strconv.Atoi(fields["viewport_width_css"])
	vh, err2 := strconv.Atoi(fields["viewport_height_css"])
	dpr, err3 := strconv.ParseFloat(fields["device_pixel_ratio"], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		s.fail(w, r, domain.Invalid("INVALID_CAPTURE", "viewport_width_css, viewport_height_css, and device_pixel_ratio must be numbers"))
		return
	}

	a, err := s.svc.IngestAsset(r.Context(), application.IngestAssetInput{
		ProjectID: r.PathValue("id"),
		Role:      domain.ImageRole(fields["role"]),
		Capture: domain.Capture{
			ViewportWidthCSS: vw, ViewportHeightCSS: vh,
			DevicePixelRatio: dpr, ContentStateKey: fields["content_state_key"],
		},
		Data: data,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAsset(a))
}

func uploadReadError(err error) error {
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return domain.TooLarge("IMAGE_TOO_LARGE", "upload must be at most %d MB", domain.MaxImageBytes>>20)
	}
	return domain.Invalid("INVALID_UPLOAD", "upload could not be read")
}

func (s *Server) assetContent(w http.ResponseWriter, r *http.Request) {
	a, data, err := s.svc.AssetContent(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", a.MIME)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	// Content is addressed by an immutable asset ID.
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("ETag", `"`+a.SHA256+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) createComparison(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ReferenceAssetID string `json:"reference_asset_id"`
		ActualAssetID    string `json:"actual_asset_id"`
		Requirements     string `json:"requirements"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	cd, err := s.svc.CreateComparison(r.Context(), application.CreateComparisonInput{
		ProjectID: r.PathValue("id"), ReferenceAssetID: body.ReferenceAssetID,
		ActualAssetID: body.ActualAssetID, Requirements: body.Requirements,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toComparisonDetail(cd))
}

func (s *Server) getComparison(w http.ResponseWriter, r *http.Request) {
	cd, err := s.svc.GetComparison(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toComparisonDetail(cd))
}

func (s *Server) startRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AllowDuplicate bool `json:"allow_duplicate"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(w, r, &body); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	run, created, err := s.svc.StartRun(r.Context(), application.StartRunInput{
		ComparisonID:   r.PathValue("id"),
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
		AllowDuplicate: body.AllowDuplicate,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	status := http.StatusAccepted
	if !created {
		status = http.StatusOK
	}
	writeJSON(w, status, toRun(run))
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	rd, err := s.svc.GetRun(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, runDetailDTO{runDTO: toRun(rd.Run), Findings: toFindings(rd.Findings, rd.Reviews)})
}

func (s *Server) exportRun(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format != "json" && format != "markdown" {
		s.fail(w, r, domain.Invalid("INVALID_FORMAT", "format must be json or markdown"))
		return
	}
	e, err := s.svc.ExportRun(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	name := "framecheck-run-" + e.Run.ID
	if format == "markdown" {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.md"`)
		_, _ = io.WriteString(w, e.Markdown())
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.json"`)
	writeJSON(w, http.StatusOK, toExport(e))
}

func (s *Server) reviewFinding(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	rv, err := s.svc.ReviewFinding(r.Context(), r.PathValue("id"), domain.Decision(body.Decision), body.Reason)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toReview(rv))
}
