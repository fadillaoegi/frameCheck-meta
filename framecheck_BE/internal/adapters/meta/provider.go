// Package meta calls the hosted Meta Model API (Muse Spark) for visual
// analysis. Its output is untrusted: the application validates every finding.
// Every call may be billed, so the adapter enforces a per-process request cap
// in addition to whatever the account allows.
package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"framecheck/internal/domain"
	"framecheck/internal/ports"
)

const (
	maxResponseBytes = 4 << 20
	maxFindings      = 30
	// Boxes slightly outside 0..1000 (the docs mention e.g. 1001) are clamped;
	// anything further out is passed through and rejected by the domain.
	clampTolerance = 10
)

type Config struct {
	BaseURL         string
	APIKey          string
	Model           string
	MaxOutputTokens int
	// MaxImageSide downscales images whose longer side is larger.
	MaxImageSide int
	// MaxRequests caps provider calls for the life of the process. Meta's
	// spend limit is only an alert, so this is the local hard stop.
	MaxRequests int64
	Timeout     time.Duration
	HTTPClient  *http.Client
}

type Provider struct {
	cfg  Config
	http *http.Client
	used atomic.Int64
}

func New(cfg Config) (*Provider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("MODEL_API_KEY is empty")
	}
	if !strings.HasPrefix(cfg.BaseURL, "https://") && !strings.HasPrefix(cfg.BaseURL, "http://127.0.0.1") {
		return nil, errors.New("MODEL_API_BASE_URL must use https")
	}
	if cfg.Model == "" {
		return nil, errors.New("model is empty")
	}
	if cfg.MaxOutputTokens < 16 {
		cfg.MaxOutputTokens = 4000
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 110 * time.Second
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: cfg.Timeout}
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Provider{cfg: cfg, http: hc}, nil
}

func (p *Provider) Origin() domain.RunOrigin { return domain.OriginMetaAPI }
func (p *Provider) ModelID() string          { return p.cfg.Model }
func (p *Provider) PromptVersion() string    { return PromptVersion }
func (p *Provider) MaxAnalysisSide() int     { return p.cfg.MaxImageSide }

// ConfigVersion covers every request setting that can change the answer.
func (p *Provider) ConfigVersion() string {
	return fmt.Sprintf("t0-max%d-side%d", p.cfg.MaxOutputTokens, p.cfg.MaxImageSide)
}

// RequestsUsed reports how many billable calls this process has started.
func (p *Provider) RequestsUsed() int64 { return p.used.Load() }

func providerErr(code, msg string) error { return &ports.ProviderError{Code: code, Message: msg} }

func (p *Provider) Analyze(ctx context.Context, in ports.AnalysisInput) (ports.AnalysisOutput, error) {
	refURL, err := dataURL(in.Reference)
	if err != nil {
		return ports.AnalysisOutput{}, err
	}
	actURL, err := dataURL(in.Actual)
	if err != nil {
		return ports.AnalysisOutput{}, err
	}
	body, err := json.Marshal(p.request(buildPrompt(in.Requirements), refURL, actURL))
	if err != nil {
		return ports.AnalysisOutput{}, err
	}

	// Reserve budget before sending; a reserved call counts even if it fails.
	if n := p.used.Add(1); p.cfg.MaxRequests > 0 && n > p.cfg.MaxRequests {
		p.used.Add(-1)
		return ports.AnalysisOutput{}, providerErr("REQUEST_CAP_REACHED",
			fmt.Sprintf("this server already made its limit of %d model requests; raise MODEL_API_MAX_REQUESTS and restart deliberately", p.cfg.MaxRequests))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return ports.AnalysisOutput{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := p.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ports.AnalysisOutput{}, ctx.Err()
		}
		// Transport errors can include the URL but never headers; still keep
		// them out of stored messages.
		return ports.AnalysisOutput{}, providerErr("PROVIDER_UNREACHABLE", "the model API could not be reached")
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil {
		return ports.AnalysisOutput{}, providerErr("PROVIDER_UNREACHABLE", "the model API response could not be read")
	}
	requestID := res.Header.Get("X-Request-Id")
	if res.StatusCode != http.StatusOK {
		return ports.AnalysisOutput{ProviderRequestID: requestID}, httpError(res.StatusCode, raw)
	}
	if len(raw) > maxResponseBytes {
		return ports.AnalysisOutput{ProviderRequestID: requestID}, providerErr("INVALID_OUTPUT", "the model API response was too large")
	}
	out, err := parseResponse(raw)
	if out.ProviderRequestID == "" {
		out.ProviderRequestID = requestID
	}
	return out, err
}

func (p *Provider) request(prompt, refURL, actURL string) map[string]any {
	return map[string]any{
		"model": p.cfg.Model,
		"input": []any{map[string]any{
			"type": "message",
			"role": "user",
			"content": []any{
				map[string]any{"type": "input_text", "text": prompt},
				map[string]any{"type": "input_text", "text": "IMAGE 1 — REFERENCE design:"},
				map[string]any{"type": "input_image", "image_url": refURL},
				map[string]any{"type": "input_text", "text": "IMAGE 2 — ACTUAL implementation:"},
				map[string]any{"type": "input_image", "image_url": actURL},
			},
		}},
		"text": map[string]any{"format": map[string]any{
			"type":   "json_schema",
			"name":   "framecheck_findings",
			"schema": findingsSchema(),
			"strict": true,
		}},
		"temperature":       0,
		"max_output_tokens": p.cfg.MaxOutputTokens,
		"store":             false,
	}
}

// httpError maps a non-200 status to a safe, actionable failure. The body may
// echo request content, so only a short machine code from it is kept.
func httpError(status int, raw []byte) error {
	var e struct {
		Error struct {
			Code string `json:"code"`
			Type string `json:"type"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &e)
	detail := safeCode(e.Error.Code)
	if detail == "" {
		detail = safeCode(e.Error.Type)
	}
	suffix := " (HTTP " + strconv.Itoa(status)
	if detail != "" {
		suffix += ", " + detail
	}
	suffix += ")"
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return providerErr("PROVIDER_AUTH", "the model API rejected the credentials or this model is not enabled for the account"+suffix)
	case status == http.StatusPaymentRequired:
		return providerErr("PROVIDER_BILLING", "the model API refused the request for billing reasons; check credits and payment in Billing"+suffix)
	case status == http.StatusTooManyRequests:
		return providerErr("PROVIDER_RATE_LIMITED", "the model API rate limit was reached; wait before starting another run"+suffix)
	case status == http.StatusBadRequest || status == http.StatusRequestEntityTooLarge || status == http.StatusUnprocessableEntity:
		return providerErr("PROVIDER_REJECTED", "the model API rejected the request"+suffix)
	case status >= 500:
		return providerErr("PROVIDER_UNAVAILABLE", "the model API had a server error"+suffix)
	}
	return providerErr("PROVIDER_ERROR", "the model API returned an unexpected status"+suffix)
}

// safeCode keeps only short identifier-like codes such as rate_limit_exceeded.
func safeCode(s string) string {
	if len(s) == 0 || len(s) > 64 {
		return ""
	}
	for _, c := range s {
		if !(c == '_' || c == '-' || c == '.' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return ""
		}
	}
	return s
}

type apiResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Usage *struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Error *struct {
		Code string `json:"code"`
	} `json:"error"`
}

type modelBox struct {
	XMin *float64 `json:"x_min"`
	YMin *float64 `json:"y_min"`
	XMax *float64 `json:"x_max"`
	YMax *float64 `json:"y_max"`
}

type modelEvidence struct {
	Box         *modelBox `json:"box"`
	Description string    `json:"description"`
}

type modelOutput struct {
	Findings []struct {
		Category  string         `json:"category"`
		Summary   string         `json:"summary"`
		CheckStep string         `json:"check_step"`
		Reference *modelEvidence `json:"reference"`
		Actual    *modelEvidence `json:"actual"`
	} `json:"findings"`
	InspectedBothImages *bool `json:"inspected_both_images"`
}

func parseResponse(raw []byte) (ports.AnalysisOutput, error) {
	var r apiResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return ports.AnalysisOutput{}, providerErr("INVALID_OUTPUT", "the model API response was not valid JSON")
	}
	out := ports.AnalysisOutput{ProviderRequestID: r.ID}
	if r.Usage != nil {
		out.InputTokens, out.OutputTokens = r.Usage.InputTokens, r.Usage.OutputTokens
	}
	switch r.Status {
	case "completed", "":
	case "incomplete":
		reason := ""
		if r.IncompleteDetails != nil {
			reason = safeCode(r.IncompleteDetails.Reason)
		}
		return out, providerErr("INCOMPLETE_OUTPUT", "the model stopped before finishing its answer"+parens(reason))
	default:
		code := ""
		if r.Error != nil {
			code = safeCode(r.Error.Code)
		}
		return out, providerErr("PROVIDER_FAILED", "the model response has status "+safeCode(r.Status)+parens(code))
	}

	var text strings.Builder
	for _, item := range r.Output {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			if c.Type == "output_text" {
				text.WriteString(c.Text)
			}
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return out, providerErr("EMPTY_OUTPUT", "the model returned no answer text")
	}

	var m modelOutput
	dec := json.NewDecoder(strings.NewReader(text.String()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil || m.InspectedBothImages == nil {
		return out, providerErr("INVALID_OUTPUT", "the model answer did not match the findings schema")
	}

	out.Complete = *m.InspectedBothImages
	findings := m.Findings
	if len(findings) > maxFindings {
		findings = findings[:maxFindings]
		out.Complete = false
	}
	for _, f := range findings {
		out.Findings = append(out.Findings, domain.RawFinding{
			Category:  f.Category,
			Summary:   f.Summary,
			CheckStep: f.CheckStep,
			Reference: toRaw(f.Reference, "reference"),
			Actual:    toRaw(f.Actual, "actual"),
		})
	}
	return out, nil
}

func parens(s string) string {
	if s == "" {
		return ""
	}
	return " (" + s + ")"
}

func toRaw(e *modelEvidence, role string) *domain.RawEvidence {
	if e == nil {
		return nil
	}
	ev := &domain.RawEvidence{ImageRole: role, Description: e.Description}
	if b := e.Box; b != nil && b.XMin != nil && b.YMin != nil && b.XMax != nil && b.YMax != nil {
		ev.Box = &domain.NormalizedBox{X0: clamp(*b.XMin), Y0: clamp(*b.YMin), X1: clamp(*b.XMax), Y1: clamp(*b.YMax)}
	}
	return ev
}

func clamp(v float64) float64 {
	if v < 0 && v >= -clampTolerance {
		return 0
	}
	if v > domain.NormalizedScale && v <= domain.NormalizedScale+clampTolerance {
		return domain.NormalizedScale
	}
	return v
}
