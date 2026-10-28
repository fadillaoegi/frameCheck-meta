// Package application implements FrameCheck use cases on top of ports.
package application

import (
	"context"
	"log/slog"

	"framecheck/internal/domain"
	"framecheck/internal/ports"
)

type Service struct {
	repo     ports.Repository
	store    ports.AssetStore
	provider ports.InferenceProvider
	clock    ports.Clock
	ids      ports.IDGenerator
	log      *slog.Logger
	queue    chan string
}

type Deps struct {
	Repo     ports.Repository
	Store    ports.AssetStore
	Provider ports.InferenceProvider
	Clock    ports.Clock
	IDs      ports.IDGenerator
	Log      *slog.Logger
}

// queueSize bounds pending runs; one active run per project keeps it small.
const queueSize = 64

func NewService(d Deps) *Service {
	log := d.Log
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		repo: d.Repo, store: d.Store, provider: d.Provider,
		clock: d.Clock, ids: d.IDs, log: log,
		queue: make(chan string, queueSize),
	}
}

func (s *Service) CreateProject(ctx context.Context, name string) (domain.Project, error) {
	name, err := domain.NormalizeProjectName(name)
	if err != nil {
		return domain.Project{}, err
	}
	p := domain.Project{ID: s.ids.NewID(), Name: name, CreatedAt: s.clock.Now()}
	if err := s.repo.CreateProject(ctx, p); err != nil {
		return domain.Project{}, err
	}
	return p, nil
}

func (s *Service) ListProjects(ctx context.Context) ([]domain.Project, error) {
	return s.repo.ListProjects(ctx)
}

type ProjectDetail struct {
	Project     domain.Project
	Assets      []domain.Asset
	Comparisons []domain.Comparison
}

func (s *Service) GetProject(ctx context.Context, id string) (ProjectDetail, error) {
	p, err := s.repo.GetProject(ctx, id)
	if err != nil {
		return ProjectDetail{}, err
	}
	assets, err := s.repo.ListAssets(ctx, id)
	if err != nil {
		return ProjectDetail{}, err
	}
	comps, err := s.repo.ListComparisons(ctx, id)
	if err != nil {
		return ProjectDetail{}, err
	}
	return ProjectDetail{Project: p, Assets: assets, Comparisons: comps}, nil
}

// AssetContent returns stored bytes for display. Bytes are re-hashed so a
// tampered file is never served as evidence.
func (s *Service) AssetContent(ctx context.Context, id string) (domain.Asset, []byte, error) {
	a, err := s.repo.GetAsset(ctx, id)
	if err != nil {
		return domain.Asset{}, nil, err
	}
	data, err := s.loadAsset(ctx, a)
	if err != nil {
		return domain.Asset{}, nil, err
	}
	return a, data, nil
}

func (s *Service) loadAsset(ctx context.Context, a domain.Asset) ([]byte, error) {
	data, err := s.store.Get(ctx, a.StorageKey)
	if err != nil {
		return nil, err
	}
	if domain.SHA256Hex(data) != a.SHA256 {
		return nil, domain.Conflict("ASSET_INTEGRITY", "stored image no longer matches its recorded hash")
	}
	return data, nil
}
