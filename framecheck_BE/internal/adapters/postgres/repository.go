// Package postgres implements ports.Repository with pgx.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"framecheck/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const uniqueViolation = "23505"

func notFound(what string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.NotFound("NOT_FOUND", "%s not found", what)
	}
	return err
}

// isUUID avoids sending malformed IDs to Postgres, where they would surface
// as a type error instead of a clean 404.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

// ---- projects

func (r *Repository) CreateProject(ctx context.Context, p domain.Project) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO projects (id, name, created_at) VALUES ($1, $2, $3)`,
		p.ID, p.Name, p.CreatedAt)
	return err
}

func (r *Repository) ListProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text, name, created_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Project, error) {
		var p domain.Project
		err := row.Scan(&p.ID, &p.Name, &p.CreatedAt)
		return p, err
	})
}

func (r *Repository) GetProject(ctx context.Context, id string) (domain.Project, error) {
	var p domain.Project
	if !isUUID(id) {
		return p, notFound("project", pgx.ErrNoRows)
	}
	err := r.pool.QueryRow(ctx, `SELECT id::text, name, created_at FROM projects WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.CreatedAt)
	return p, notFound("project", err)
}

// ---- assets

const assetColumns = `id::text, project_id::text, role, sha256, mime, bytes, width_px, height_px,
	viewport_width_css, viewport_height_css, device_pixel_ratio, content_state_key, storage_key, created_at`

const assetInsertColumns = `id, project_id, role, sha256, mime, bytes, width_px, height_px,
	viewport_width_css, viewport_height_css, device_pixel_ratio, content_state_key, storage_key, created_at`

func scanAsset(row pgx.Row) (domain.Asset, error) {
	var a domain.Asset
	var role string
	err := row.Scan(&a.ID, &a.ProjectID, &role, &a.SHA256, &a.MIME, &a.Bytes, &a.WidthPx, &a.HeightPx,
		&a.Capture.ViewportWidthCSS, &a.Capture.ViewportHeightCSS, &a.Capture.DevicePixelRatio,
		&a.Capture.ContentStateKey, &a.StorageKey, &a.CreatedAt)
	a.Role = domain.ImageRole(role)
	return a, err
}

func (r *Repository) CreateAsset(ctx context.Context, a domain.Asset) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO assets (`+assetInsertColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		a.ID, a.ProjectID, a.Role, a.SHA256, a.MIME, a.Bytes, a.WidthPx, a.HeightPx,
		a.Capture.ViewportWidthCSS, a.Capture.ViewportHeightCSS, a.Capture.DevicePixelRatio,
		a.Capture.ContentStateKey, a.StorageKey, a.CreatedAt)
	return err
}

func (r *Repository) GetAsset(ctx context.Context, id string) (domain.Asset, error) {
	if !isUUID(id) {
		return domain.Asset{}, notFound("asset", pgx.ErrNoRows)
	}
	a, err := scanAsset(r.pool.QueryRow(ctx, `SELECT `+assetColumns+` FROM assets WHERE id = $1`, id))
	return a, notFound("asset", err)
}

func (r *Repository) ListAssets(ctx context.Context, projectID string) ([]domain.Asset, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+assetColumns+` FROM assets WHERE project_id = $1 ORDER BY created_at`, projectID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Asset, error) { return scanAsset(row) })
}

// ---- requirements

const requirementColumns = `id::text, project_id::text, version, text, sha256, created_at`

func scanRequirement(row pgx.Row) (domain.RequirementVersion, error) {
	var q domain.RequirementVersion
	err := row.Scan(&q.ID, &q.ProjectID, &q.Version, &q.Text, &q.SHA256, &q.CreatedAt)
	return q, err
}

func (r *Repository) LatestRequirement(ctx context.Context, projectID string) (*domain.RequirementVersion, error) {
	q, err := scanRequirement(r.pool.QueryRow(ctx, `SELECT `+requirementColumns+`
		FROM requirement_versions WHERE project_id = $1 ORDER BY version DESC LIMIT 1`, projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *Repository) CreateRequirement(ctx context.Context, q domain.RequirementVersion) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO requirement_versions (id, project_id, version, text, sha256, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`, q.ID, q.ProjectID, q.Version, q.Text, q.SHA256, q.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return domain.Conflict("REQUIREMENT_CONFLICT", "requirements changed concurrently; try again")
	}
	return err
}

func (r *Repository) GetRequirement(ctx context.Context, id string) (domain.RequirementVersion, error) {
	q, err := scanRequirement(r.pool.QueryRow(ctx, `SELECT `+requirementColumns+` FROM requirement_versions WHERE id = $1`, id))
	return q, notFound("requirement version", err)
}

// ---- comparisons

const comparisonColumns = `id::text, project_id::text, reference_asset_id::text, actual_asset_id::text,
	requirement_version_id::text, created_at`

func scanComparison(row pgx.Row) (domain.Comparison, error) {
	var c domain.Comparison
	err := row.Scan(&c.ID, &c.ProjectID, &c.ReferenceAssetID, &c.ActualAssetID, &c.RequirementVersionID, &c.CreatedAt)
	return c, err
}

func (r *Repository) CreateComparison(ctx context.Context, c domain.Comparison) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO comparisons (id, project_id, reference_asset_id, actual_asset_id, requirement_version_id, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		c.ID, c.ProjectID, c.ReferenceAssetID, c.ActualAssetID, c.RequirementVersionID, c.CreatedAt)
	return err
}

func (r *Repository) GetComparison(ctx context.Context, id string) (domain.Comparison, error) {
	if !isUUID(id) {
		return domain.Comparison{}, notFound("comparison", pgx.ErrNoRows)
	}
	c, err := scanComparison(r.pool.QueryRow(ctx, `SELECT `+comparisonColumns+` FROM comparisons WHERE id = $1`, id))
	return c, notFound("comparison", err)
}

func (r *Repository) ListComparisons(ctx context.Context, projectID string) ([]domain.Comparison, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+comparisonColumns+` FROM comparisons WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Comparison, error) { return scanComparison(row) })
}

// ---- runs

const runColumns = `id::text, project_id::text, comparison_id::text, status, origin, input_fingerprint,
	coalesce(idempotency_key, ''), model_id, config_version, prompt_version, transform,
	provider_request_id, latency_ms, input_tokens, output_tokens, failure_code, failure_message, issues,
	created_at, started_at, finished_at`

func scanRun(row pgx.Row) (domain.Run, error) {
	var run domain.Run
	var status, origin string
	var transform, issues []byte
	err := row.Scan(&run.ID, &run.ProjectID, &run.ComparisonID, &status, &origin, &run.InputFingerprint,
		&run.IdempotencyKey, &run.ModelID, &run.ConfigVersion, &run.PromptVersion, &transform,
		&run.ProviderRequestID, &run.LatencyMS, &run.InputTokens, &run.OutputTokens, &run.FailureCode, &run.FailureMessage, &issues,
		&run.CreatedAt, &run.StartedAt, &run.FinishedAt)
	if err != nil {
		return run, err
	}
	run.Status = domain.RunStatus(status)
	run.Origin = domain.RunOrigin(origin)
	if err := json.Unmarshal(transform, &run.Transform); err != nil {
		return run, err
	}
	return run, json.Unmarshal(issues, &run.Issues)
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func jsonList(items []string) []byte {
	if items == nil {
		items = []string{}
	}
	b, _ := json.Marshal(items)
	return b
}

func (r *Repository) CreateRun(ctx context.Context, run domain.Run) error {
	transform, err := json.Marshal(run.Transform)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO runs (id, project_id, comparison_id, status, origin,
		input_fingerprint, idempotency_key, model_id, config_version, prompt_version, transform, issues, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		run.ID, run.ProjectID, run.ComparisonID, run.Status, run.Origin, run.InputFingerprint,
		nullable(run.IdempotencyKey), run.ModelID, run.ConfigVersion, run.PromptVersion, transform,
		jsonList(run.Issues), run.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		switch pgErr.ConstraintName {
		case "runs_one_active_per_project":
			return domain.Conflict("RUN_ACTIVE", "this project already has an analysis queued or running")
		case "runs_idempotency":
			return domain.Conflict("IDEMPOTENCY_CONFLICT", "this Idempotency-Key is already in use")
		}
	}
	return err
}

func (r *Repository) GetRun(ctx context.Context, id string) (domain.Run, error) {
	if !isUUID(id) {
		return domain.Run{}, notFound("run", pgx.ErrNoRows)
	}
	run, err := scanRun(r.pool.QueryRow(ctx, `SELECT `+runColumns+` FROM runs WHERE id = $1`, id))
	return run, notFound("run", err)
}

func (r *Repository) ListRuns(ctx context.Context, comparisonID string) ([]domain.Run, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+runColumns+` FROM runs WHERE comparison_id = $1 ORDER BY created_at DESC`, comparisonID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Run, error) { return scanRun(row) })
}

func (r *Repository) optionalRun(row pgx.Row) (*domain.Run, error) {
	run, err := scanRun(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *Repository) FindRunByIdempotencyKey(ctx context.Context, comparisonID, key string) (*domain.Run, error) {
	return r.optionalRun(r.pool.QueryRow(ctx, `SELECT `+runColumns+` FROM runs
		WHERE comparison_id = $1 AND idempotency_key = $2`, comparisonID, key))
}

func (r *Repository) FindReusableRun(ctx context.Context, fingerprint string) (*domain.Run, error) {
	return r.optionalRun(r.pool.QueryRow(ctx, `SELECT `+runColumns+` FROM runs
		WHERE input_fingerprint = $1 AND status IN ('completed', 'needs_review')
		ORDER BY created_at DESC LIMIT 1`, fingerprint))
}

func (r *Repository) MarkRunRunning(ctx context.Context, id string, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE runs SET status = 'running', started_at = $2
		WHERE id = $1 AND status = 'queued'`, id, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return domain.Conflict("RUN_NOT_QUEUED", "run is no longer queued")
	}
	return nil
}

type boxJSON struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type evidenceJSON struct {
	AssetID     string   `json:"asset_id"`
	ImageRole   string   `json:"image_role"`
	Box         *boxJSON `json:"box_px"`
	Description string   `json:"description"`
}

func encodeEvidence(ev *domain.Evidence) ([]byte, error) {
	if ev == nil {
		return nil, nil
	}
	e := evidenceJSON{AssetID: ev.AssetID, ImageRole: string(ev.ImageRole), Description: ev.Description}
	if ev.Box != nil {
		e.Box = &boxJSON{X: ev.Box.X, Y: ev.Box.Y, Width: ev.Box.Width, Height: ev.Box.Height}
	}
	return json.Marshal(e)
}

func decodeEvidence(b []byte) (*domain.Evidence, error) {
	if b == nil {
		return nil, nil
	}
	var e evidenceJSON
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, err
	}
	ev := &domain.Evidence{AssetID: e.AssetID, ImageRole: domain.ImageRole(e.ImageRole), Description: e.Description}
	if e.Box != nil {
		ev.Box = &domain.Box{X: e.Box.X, Y: e.Box.Y, Width: e.Box.Width, Height: e.Box.Height}
	}
	return ev, nil
}

func (r *Repository) FinishRun(ctx context.Context, run domain.Run, findings []domain.Finding) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE runs SET status = $2, provider_request_id = $3, latency_ms = $4,
			failure_code = $5, failure_message = $6, issues = $7, started_at = $8, finished_at = $9,
			input_tokens = $10, output_tokens = $11
			WHERE id = $1 AND status IN ('queued', 'running')`,
			run.ID, run.Status, run.ProviderRequestID, run.LatencyMS, run.FailureCode, run.FailureMessage,
			jsonList(run.Issues), run.StartedAt, run.FinishedAt, run.InputTokens, run.OutputTokens)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return domain.Conflict("RUN_ALREADY_FINISHED", "run already has a terminal status")
		}
		for i, f := range findings {
			ref, err := encodeEvidence(f.ReferenceEvidence)
			if err != nil {
				return err
			}
			act, err := encodeEvidence(f.ActualEvidence)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO findings (id, run_id, position, category, summary, check_step,
				reference_evidence, actual_evidence, evidence_status, issues, created_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
				f.ID, run.ID, i, f.Category, f.Summary, f.CheckStep, ref, act, f.EvidenceStatus,
				jsonList(f.Issues), f.CreatedAt); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) FailInterruptedRuns(ctx context.Context, at time.Time) (int, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE runs SET status = 'failed', failure_code = 'INTERRUPTED',
		failure_message = 'the server stopped before this analysis finished', finished_at = $1
		WHERE status IN ('queued', 'running')`, at)
	return int(tag.RowsAffected()), err
}

// ---- findings and reviews

const findingColumns = `id::text, run_id::text, category, summary, check_step, reference_evidence,
	actual_evidence, evidence_status, issues, created_at`

func scanFinding(row pgx.Row) (domain.Finding, error) {
	var f domain.Finding
	var category, status string
	var ref, act, issues []byte
	if err := row.Scan(&f.ID, &f.RunID, &category, &f.Summary, &f.CheckStep, &ref, &act, &status, &issues, &f.CreatedAt); err != nil {
		return f, err
	}
	f.Category = domain.Category(category)
	f.EvidenceStatus = domain.EvidenceStatus(status)
	var err error
	if f.ReferenceEvidence, err = decodeEvidence(ref); err != nil {
		return f, err
	}
	if f.ActualEvidence, err = decodeEvidence(act); err != nil {
		return f, err
	}
	return f, json.Unmarshal(issues, &f.Issues)
}

func (r *Repository) GetFinding(ctx context.Context, id string) (domain.Finding, error) {
	if !isUUID(id) {
		return domain.Finding{}, notFound("finding", pgx.ErrNoRows)
	}
	f, err := scanFinding(r.pool.QueryRow(ctx, `SELECT `+findingColumns+` FROM findings WHERE id = $1`, id))
	return f, notFound("finding", err)
}

func (r *Repository) ListFindings(ctx context.Context, runID string) ([]domain.Finding, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+findingColumns+` FROM findings WHERE run_id = $1 ORDER BY position`, runID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Finding, error) { return scanFinding(row) })
}

func (r *Repository) CreateReview(ctx context.Context, rv domain.Review) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO reviews (id, finding_id, decision, reason, reviewed_at)
		VALUES ($1,$2,$3,$4,$5)`, rv.ID, rv.FindingID, rv.Decision, rv.Reason, rv.ReviewedAt)
	return err
}

func (r *Repository) ListReviewsForRun(ctx context.Context, runID string) ([]domain.Review, error) {
	rows, err := r.pool.Query(ctx, `SELECT rv.id::text, rv.finding_id::text, rv.decision, rv.reason, rv.reviewed_at
		FROM reviews rv JOIN findings f ON f.id = rv.finding_id
		WHERE f.run_id = $1 ORDER BY rv.reviewed_at, rv.id`, runID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Review, error) {
		var rv domain.Review
		var decision string
		err := row.Scan(&rv.ID, &rv.FindingID, &decision, &rv.Reason, &rv.ReviewedAt)
		rv.Decision = domain.Decision(decision)
		return rv, err
	})
}
