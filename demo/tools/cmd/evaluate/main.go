// Command evaluate runs every fixture case through a running FrameCheck API and
// scores the findings against the manifest ground truth. It measures whatever
// provider the API is configured with and records that provider's origin; a
// mock or fixture score is a pipeline check, never a model result.
//
//	go run ./cmd/evaluate -api http://127.0.0.1:8089     (from demo/tools)
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"framecheck/demo/tools/internal/fixtures"
)

var categories = []string{"missing_element", "clipped_text", "layout_mismatch"}

func main() {
	api := flag.String("api", "http://127.0.0.1:8089", "FrameCheck API base URL")
	demoDir := flag.String("demo", "..", "path to the demo/ directory")
	threshold := flag.Float64("iou", 0.5, "minimum IoU for a finding to match a labelled region")
	split := flag.String("split", "dev", "case split to evaluate")
	fixturesName := flag.String("fixtures", "fixtures", "fixture directory relative to demo/ (fixtures-holdout for the holdout split)")
	flag.Parse()
	if err := run(strings.TrimRight(*api, "/"), *demoDir, *fixturesName, *threshold, *split); err != nil {
		log.Fatal(err)
	}
}

// ---- API client

type client struct {
	base string
	http *http.Client
}

type apiError struct {
	Error struct{ Code, Message string } `json:"error"`
}

func (c *client) do(req *http.Request, out any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		var e apiError
		_ = json.Unmarshal(body, &e)
		return fmt.Errorf("%s %s: %d %s %s", req.Method, req.URL.Path, res.StatusCode, e.Error.Code, e.Error.Message)
	}
	return json.Unmarshal(body, out)
}

func (c *client) postJSON(path string, in, out any, headers map[string]string) error {
	b, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.do(req, out)
}

func (c *client) get(path string, out any) error {
	req, _ := http.NewRequest(http.MethodGet, c.base+path, nil)
	return c.do(req, out)
}

func (c *client) upload(projectID, file, role string, cap fixtures.Capture) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("role", role)
	_ = w.WriteField("viewport_width_css", strconv.Itoa(cap.ViewportWidthCSS))
	_ = w.WriteField("viewport_height_css", strconv.Itoa(cap.ViewportHeightCSS))
	_ = w.WriteField("device_pixel_ratio", strconv.FormatFloat(cap.DevicePixelRatio, 'f', -1, 64))
	_ = w.WriteField("content_state_key", cap.ContentStateKey)
	fw, _ := w.CreateFormFile("file", filepath.Base(file))
	_, _ = fw.Write(data)
	_ = w.Close()
	req, _ := http.NewRequest(http.MethodPost, c.base+"/v1/projects/"+projectID+"/assets", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	var out struct{ ID string }
	return out.ID, c.do(req, &out)
}

type evidence struct {
	Box *fixtures.Box `json:"box_px"`
}

type finding struct {
	Category          string    `json:"category"`
	EvidenceStatus    string    `json:"evidence_status"`
	ReferenceEvidence *evidence `json:"reference_evidence"`
	ActualEvidence    *evidence `json:"actual_evidence"`
}

// box returns the evidence box compared against the labelled region.
func (f finding) box() *fixtures.Box {
	ev := f.ActualEvidence
	if f.Category == "missing_element" {
		ev = f.ReferenceEvidence
	}
	if ev == nil {
		return nil
	}
	return ev.Box
}

type runDetail struct {
	ID                string    `json:"id"`
	Status            string    `json:"status"`
	Origin            string    `json:"origin"`
	ModelID           string    `json:"model_id"`
	ConfigVersion     string    `json:"config_version"`
	PromptVersion     string    `json:"prompt_version"`
	ProviderRequestID string    `json:"provider_request_id"`
	LatencyMS         int64     `json:"latency_ms"`
	InputTokens       int64     `json:"input_tokens"`
	OutputTokens      int64     `json:"output_tokens"`
	FailureCode       string    `json:"failure_code"`
	Findings          []finding `json:"findings"`
}

// ---- scoring

type counts struct {
	TP, FP, FN int
	IoUSum     float64
}

func (c counts) Precision() *float64 { return ratio(c.TP, c.TP+c.FP) }
func (c counts) Recall() *float64    { return ratio(c.TP, c.TP+c.FN) }

func ratio(a, b int) *float64 {
	if b == 0 {
		return nil
	}
	r := float64(a) / float64(b)
	return &r
}

type caseResult struct {
	ID          string   `json:"id"`
	Category    *string  `json:"category"`
	Component   *string  `json:"component"`
	RunID       string   `json:"run_id"`
	Status      string   `json:"status"`
	FailureCode string   `json:"failure_code,omitempty"`
	LatencyMS   int64    `json:"latency_ms"`
	Findings    int      `json:"findings"`
	TP          int      `json:"tp"`
	FP          int      `json:"fp"`
	FN          int      `json:"fn"`
	BestIoU     *float64 `json:"best_iou"`
}

// score greedily matches each expected region to the unmatched finding of the
// same category with the highest IoU at or above the threshold.
func score(exp []fixtures.Expected, got []finding, thr float64) (tp, fn int, fpByCat map[string]int, ious []float64, best *float64) {
	used := make([]bool, len(got))
	fpByCat = map[string]int{}
	for _, e := range exp {
		target := e.MatchBox()
		bi, bv := -1, 0.0
		for i, f := range got {
			if used[i] || f.Category != e.Category || target == nil || f.box() == nil {
				continue
			}
			if v := f.box().IoU(*target); v > bv {
				bi, bv = i, v
			}
		}
		if best == nil || bv > *best {
			b := bv
			best = &b
		}
		if bi >= 0 && bv >= thr {
			used[bi] = true
			tp++
			ious = append(ious, bv)
		} else {
			fn++
		}
	}
	for i, f := range got {
		if !used[i] {
			fpByCat[f.Category]++
		}
	}
	return
}

type report struct {
	GeneratedAt   string                 `json:"generated_at"`
	Note          string                 `json:"note"`
	API           string                 `json:"api"`
	Split         string                 `json:"split"`
	IoUThreshold  float64                `json:"iou_threshold"`
	Origin        string                 `json:"origin"`
	ModelID       string                 `json:"model_id"`
	ConfigVersion string                 `json:"config_version"`
	PromptVersion string                 `json:"prompt_version"`
	ManifestPage  string                 `json:"manifest_page_sha256"`
	ProjectID     string                 `json:"project_id"`
	Runs          int                    `json:"runs"`
	Failed        int                    `json:"failed"`
	NeedsReview   int                    `json:"needs_review"`
	ControlFP     int                    `json:"control_false_positives"`
	InputTokens   int64                  `json:"input_tokens_total"`
	OutputTokens  int64                  `json:"output_tokens_total"`
	PerCategory   map[string]categoryOut `json:"per_category"`
	Overall       categoryOut            `json:"overall"`
	LatencyMS     latency                `json:"latency_ms"`
	Cases         []caseResult           `json:"cases"`
}

type categoryOut struct {
	TP        int      `json:"tp"`
	FP        int      `json:"fp"`
	FN        int      `json:"fn"`
	Precision *float64 `json:"precision"`
	Recall    *float64 `json:"recall"`
	MeanIoU   *float64 `json:"mean_iou_of_matches"`
}

type latency struct {
	Min, Median, Max int64
}

func summarize(c counts) categoryOut {
	out := categoryOut{TP: c.TP, FP: c.FP, FN: c.FN, Precision: c.Precision(), Recall: c.Recall()}
	if c.TP > 0 {
		m := c.IoUSum / float64(c.TP)
		out.MeanIoU = &m
	}
	return out
}

// ---- main flow

func run(api, demoDir, fixturesName string, thr float64, split string) error {
	var m fixtures.Manifest
	fixDir := filepath.Join(demoDir, fixturesName)
	if err := fixtures.ReadJSON(filepath.Join(fixDir, "manifest.json"), &m); err != nil {
		return err
	}
	c := &client{base: api, http: &http.Client{Timeout: 60 * time.Second}}

	stamp := time.Now().UTC().Format("20060102-150405")
	var project struct{ ID string }
	if err := c.postJSON("/v1/projects", map[string]string{"name": "Evaluation " + stamp}, &project, nil); err != nil {
		return err
	}
	refID, err := c.upload(project.ID, filepath.Join(fixDir, m.Reference.File), "reference", m.Capture)
	if err != nil {
		return err
	}

	rep := report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339), API: api, Split: split, IoUThreshold: thr,
		ManifestPage: m.PageSHA256, ProjectID: project.ID, PerCategory: map[string]categoryOut{},
	}
	per := map[string]*counts{}
	for _, cat := range categories {
		per[cat] = &counts{}
	}
	var lat []int64

	for _, cs := range m.Cases {
		if cs.Split != split {
			continue
		}
		actID, err := c.upload(project.ID, filepath.Join(fixDir, cs.Actual.File), "actual", m.Capture)
		if err != nil {
			return fmt.Errorf("%s: %w", cs.ID, err)
		}
		var cmp struct{ ID string }
		if err := c.postJSON("/v1/projects/"+project.ID+"/comparisons", map[string]string{
			"reference_asset_id": refID, "actual_asset_id": actID,
			"requirements": "Check for missing elements, clipped text, and layout mismatches.",
		}, &cmp, nil); err != nil {
			return fmt.Errorf("%s: %w", cs.ID, err)
		}
		var started struct{ ID string }
		if err := c.postJSON("/v1/comparisons/"+cmp.ID+"/runs", map[string]bool{"allow_duplicate": true}, &started,
			map[string]string{"Idempotency-Key": "eval-" + stamp + "-" + cs.ID + "-" + nonce()}); err != nil {
			return fmt.Errorf("%s: %w", cs.ID, err)
		}
		rd, err := waitRun(c, started.ID)
		if err != nil {
			return fmt.Errorf("%s: %w", cs.ID, err)
		}
		if rep.Origin == "" {
			rep.Origin, rep.ModelID, rep.ConfigVersion, rep.PromptVersion = rd.Origin, rd.ModelID, rd.ConfigVersion, rd.PromptVersion
		}
		rep.Runs++
		rep.InputTokens += rd.InputTokens
		rep.OutputTokens += rd.OutputTokens

		cr := caseResult{ID: cs.ID, Category: cs.Category, Component: cs.Component, RunID: rd.ID, Status: rd.Status,
			FailureCode: rd.FailureCode, LatencyMS: rd.LatencyMS, Findings: len(rd.Findings)}
		switch rd.Status {
		case "failed":
			rep.Failed++
			// A failed run found nothing; every labelled defect is missed.
			cr.FN = len(cs.Expected)
			for _, e := range cs.Expected {
				per[e.Category].FN++
			}
		default:
			if rd.Status == "needs_review" {
				rep.NeedsReview++
			}
			lat = append(lat, rd.LatencyMS)
			tp, fn, fps, ious, best := score(cs.Expected, rd.Findings, thr)
			cr.TP, cr.FN, cr.BestIoU = tp, fn, best
			if len(cs.Expected) > 0 {
				cat := cs.Expected[0].Category
				per[cat].TP += tp
				per[cat].FN += fn
				for _, v := range ious {
					per[cat].IoUSum += v
				}
			}
			for cat, n := range fps {
				cr.FP += n
				if pc, ok := per[cat]; ok {
					pc.FP += n
				}
			}
			if cs.Category == nil {
				rep.ControlFP += cr.FP
			}
		}
		rep.Cases = append(rep.Cases, cr)
		log.Printf("%-22s %-12s findings=%d tp=%d fp=%d fn=%d", cs.ID, cr.Status, cr.Findings, cr.TP, cr.FP, cr.FN)
	}

	var all counts
	for _, cat := range categories {
		pc := per[cat]
		rep.PerCategory[cat] = summarize(*pc)
		all.TP += pc.TP
		all.FP += pc.FP
		all.FN += pc.FN
		all.IoUSum += pc.IoUSum
	}
	rep.Overall = summarize(all)
	if len(lat) > 0 {
		sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
		rep.LatencyMS = latency{Min: lat[0], Median: lat[len(lat)/2], Max: lat[len(lat)-1]}
	}
	rep.Note = noteFor(rep.Origin)

	outDir := filepath.Join(demoDir, "results")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	base := filepath.Join(outDir, stamp+"-"+split+"-"+rep.Origin)
	if err := fixtures.WriteJSON(base+".json", rep); err != nil {
		return err
	}
	md := markdown(rep)
	if err := os.WriteFile(base+".md", []byte(md), 0o644); err != nil {
		return err
	}
	fmt.Print(md)
	log.Printf("wrote %s.{json,md}", base)
	return nil
}

func noteFor(origin string) string {
	switch origin {
	case "meta_api":
		return "Hosted model run on the development split. Not a holdout result; do not generalise beyond this synthetic page."
	case "fixture":
		return "Fixture provider replays ground truth: a perfect score only proves the pipeline and scorer agree. Not model output."
	default:
		return "Mock provider (pixel-difference heuristic): a baseline and pipeline check. Not model output."
	}
}

func waitRun(c *client, id string) (runDetail, error) {
	deadline := time.Now().Add(4 * time.Minute)
	for {
		var rd runDetail
		if err := c.get("/v1/runs/"+id, &rd); err != nil {
			return rd, err
		}
		if rd.Status != "queued" && rd.Status != "running" {
			return rd, nil
		}
		if time.Now().After(deadline) {
			return rd, fmt.Errorf("run %s still %s after 4 minutes", id, rd.Status)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func nonce() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func pct(p *float64) string {
	if p == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.0f%%", *p*100)
}

func num(p *float64) string {
	if p == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.2f", *p)
}

func markdown(r report) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# FrameCheck evaluation — %s split, origin **%s**\n\n", r.Split, r.Origin)
	w("> %s\n\n", r.Note)
	w("Generated %s · model `%s` · config `%s` · prompt `%s` · IoU ≥ %.2f · page sha256 `%s`\n\n",
		r.GeneratedAt, r.ModelID, r.ConfigVersion, r.PromptVersion, r.IoUThreshold, r.ManifestPage[:12])
	w("Runs %d · failed %d · needs_review %d · false positives on unchanged controls **%d**", r.Runs, r.Failed, r.NeedsReview, r.ControlFP)
	if r.InputTokens > 0 || r.OutputTokens > 0 {
		w(" · tokens %d in / %d out", r.InputTokens, r.OutputTokens)
	}
	if r.LatencyMS.Max > 0 {
		w(" · latency min/median/max %d/%d/%d ms", r.LatencyMS.Min, r.LatencyMS.Median, r.LatencyMS.Max)
	}
	w("\n\n| Category | TP | FP | FN | Precision | Recall | Mean IoU |\n| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, cat := range categories {
		c := r.PerCategory[cat]
		w("| %s | %d | %d | %d | %s | %s | %s |\n", cat, c.TP, c.FP, c.FN, pct(c.Precision), pct(c.Recall), num(c.MeanIoU))
	}
	o := r.Overall
	w("| **all** | %d | %d | %d | %s | %s | %s |\n\n", o.TP, o.FP, o.FN, pct(o.Precision), pct(o.Recall), num(o.MeanIoU))
	w("| Case | Expected | Status | Findings | TP | FP | FN | Best IoU |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, c := range r.Cases {
		exp := "none (control)"
		if c.Category != nil {
			exp = *c.Category + " / " + *c.Component
		}
		status := c.Status
		if c.FailureCode != "" {
			status += " (" + c.FailureCode + ")"
		}
		w("| %s | %s | %s | %d | %d | %d | %d | %s |\n", c.ID, exp, status, c.Findings, c.TP, c.FP, c.FN, num(c.BestIoU))
	}
	return b.String()
}
