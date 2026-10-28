package application

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"framecheck/internal/domain"
)

// memRepo is an in-memory ports.Repository for use-case tests.
type memRepo struct {
	mu          sync.Mutex
	projects    map[string]domain.Project
	assets      map[string]domain.Asset
	reqs        map[string]domain.RequirementVersion
	comparisons map[string]domain.Comparison
	runs        map[string]domain.Run
	findings    map[string]domain.Finding
	reviews     []domain.Review
}

func newMemRepo() *memRepo {
	return &memRepo{
		projects: map[string]domain.Project{}, assets: map[string]domain.Asset{},
		reqs: map[string]domain.RequirementVersion{}, comparisons: map[string]domain.Comparison{},
		runs: map[string]domain.Run{}, findings: map[string]domain.Finding{},
	}
}

func nf(what string) error { return domain.NotFound("NOT_FOUND", "%s not found", what) }

func (m *memRepo) CreateProject(_ context.Context, p domain.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[p.ID] = p
	return nil
}

func (m *memRepo) ListProjects(context.Context) ([]domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Project
	for _, p := range m.projects {
		out = append(out, p)
	}
	return out, nil
}

func (m *memRepo) GetProject(_ context.Context, id string) (domain.Project, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.projects[id]
	if !ok {
		return p, nf("project")
	}
	return p, nil
}

func (m *memRepo) CreateAsset(_ context.Context, a domain.Asset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assets[a.ID] = a
	return nil
}

func (m *memRepo) GetAsset(_ context.Context, id string) (domain.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.assets[id]
	if !ok {
		return a, nf("asset")
	}
	return a, nil
}

func (m *memRepo) ListAssets(_ context.Context, projectID string) ([]domain.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Asset
	for _, a := range m.assets {
		if a.ProjectID == projectID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *memRepo) LatestRequirement(_ context.Context, projectID string) (*domain.RequirementVersion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var latest *domain.RequirementVersion
	for _, r := range m.reqs {
		if r.ProjectID == projectID && (latest == nil || r.Version > latest.Version) {
			r := r
			latest = &r
		}
	}
	return latest, nil
}

func (m *memRepo) CreateRequirement(_ context.Context, r domain.RequirementVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reqs[r.ID] = r
	return nil
}

func (m *memRepo) GetRequirement(_ context.Context, id string) (domain.RequirementVersion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reqs[id]
	if !ok {
		return r, nf("requirement")
	}
	return r, nil
}

func (m *memRepo) CreateComparison(_ context.Context, c domain.Comparison) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.comparisons[c.ID] = c
	return nil
}

func (m *memRepo) GetComparison(_ context.Context, id string) (domain.Comparison, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.comparisons[id]
	if !ok {
		return c, nf("comparison")
	}
	return c, nil
}

func (m *memRepo) ListComparisons(_ context.Context, projectID string) ([]domain.Comparison, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Comparison
	for _, c := range m.comparisons {
		if c.ProjectID == projectID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *memRepo) CreateRun(_ context.Context, r domain.Run) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, other := range m.runs {
		if other.ProjectID == r.ProjectID && (other.Status == domain.RunQueued || other.Status == domain.RunRunning) {
			return domain.Conflict("RUN_ACTIVE", "active")
		}
		if r.IdempotencyKey != "" && other.ComparisonID == r.ComparisonID && other.IdempotencyKey == r.IdempotencyKey {
			return domain.Conflict("IDEMPOTENCY_CONFLICT", "dup key")
		}
	}
	m.runs[r.ID] = r
	return nil
}

func (m *memRepo) GetRun(_ context.Context, id string) (domain.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[id]
	if !ok {
		return r, nf("run")
	}
	return r, nil
}

func (m *memRepo) ListRuns(_ context.Context, comparisonID string) ([]domain.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Run
	for _, r := range m.runs {
		if r.ComparisonID == comparisonID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memRepo) FindRunByIdempotencyKey(_ context.Context, comparisonID, key string) (*domain.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.runs {
		if r.ComparisonID == comparisonID && r.IdempotencyKey == key {
			return &r, nil
		}
	}
	return nil, nil
}

func (m *memRepo) FindReusableRun(_ context.Context, fp string) (*domain.Run, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.runs {
		if r.InputFingerprint == fp && r.Status.Finished() {
			return &r, nil
		}
	}
	return nil, nil
}

func (m *memRepo) MarkRunRunning(_ context.Context, id string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.runs[id]
	if r.Status != domain.RunQueued {
		return domain.Conflict("RUN_NOT_QUEUED", "not queued")
	}
	r.Status, r.StartedAt = domain.RunRunning, &at
	m.runs[id] = r
	return nil
}

func (m *memRepo) FinishRun(_ context.Context, r domain.Run, findings []domain.Finding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := m.runs[r.ID]
	if cur.Status != domain.RunQueued && cur.Status != domain.RunRunning {
		return domain.Conflict("RUN_ALREADY_FINISHED", "finished")
	}
	m.runs[r.ID] = r
	for _, f := range findings {
		m.findings[f.ID] = f
	}
	return nil
}

func (m *memRepo) FailInterruptedRuns(_ context.Context, at time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for id, r := range m.runs {
		if r.Status == domain.RunQueued || r.Status == domain.RunRunning {
			r.Status, r.FailureCode, r.FinishedAt = domain.RunFailed, "INTERRUPTED", &at
			m.runs[id] = r
			n++
		}
	}
	return n, nil
}

func (m *memRepo) GetFinding(_ context.Context, id string) (domain.Finding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.findings[id]
	if !ok {
		return f, nf("finding")
	}
	return f, nil
}

func (m *memRepo) ListFindings(_ context.Context, runID string) ([]domain.Finding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Finding
	for _, f := range m.findings {
		if f.RunID == runID {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *memRepo) CreateReview(_ context.Context, r domain.Review) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reviews = append(m.reviews, r)
	return nil
}

func (m *memRepo) ListReviewsForRun(_ context.Context, runID string) ([]domain.Review, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Review
	for _, r := range m.reviews {
		if f, ok := m.findings[r.FindingID]; ok && f.RunID == runID {
			out = append(out, r)
		}
	}
	return out, nil
}

type memStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func (s *memStore) Put(_ context.Context, key string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = append([]byte(nil), data...)
	return nil
}

func (s *memStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.data[key]
	if !ok {
		return nil, nf("file")
	}
	return d, nil
}

type fixedClock struct{ t time.Time }

func (c *fixedClock) Now() time.Time { c.t = c.t.Add(time.Millisecond); return c.t }

type seqIDs struct{ n int }

func (g *seqIDs) NewID() string { g.n++; return fmt.Sprintf("id-%04d", g.n) }
