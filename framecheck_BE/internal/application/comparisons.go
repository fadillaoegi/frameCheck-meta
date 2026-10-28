package application

import (
	"context"

	"framecheck/internal/domain"
)

type CreateComparisonInput struct {
	ProjectID        string
	ReferenceAssetID string
	ActualAssetID    string
	Requirements     string
}

type ComparisonDetail struct {
	Comparison  domain.Comparison
	Reference   domain.Asset
	Actual      domain.Asset
	Requirement domain.RequirementVersion
	Runs        []domain.Run
}

func (s *Service) CreateComparison(ctx context.Context, in CreateComparisonInput) (ComparisonDetail, error) {
	if _, err := s.repo.GetProject(ctx, in.ProjectID); err != nil {
		return ComparisonDetail{}, err
	}
	text, err := domain.NormalizeRequirements(in.Requirements)
	if err != nil {
		return ComparisonDetail{}, err
	}
	ref, err := s.pairAsset(ctx, in.ReferenceAssetID)
	if err != nil {
		return ComparisonDetail{}, err
	}
	act, err := s.pairAsset(ctx, in.ActualAssetID)
	if err != nil {
		return ComparisonDetail{}, err
	}
	if err := domain.ValidatePair(in.ProjectID, ref, act); err != nil {
		return ComparisonDetail{}, err
	}

	req, err := s.requirementVersion(ctx, in.ProjectID, text)
	if err != nil {
		return ComparisonDetail{}, err
	}
	c := domain.Comparison{
		ID:                   s.ids.NewID(),
		ProjectID:            in.ProjectID,
		ReferenceAssetID:     ref.ID,
		ActualAssetID:        act.ID,
		RequirementVersionID: req.ID,
		CreatedAt:            s.clock.Now(),
	}
	if err := s.repo.CreateComparison(ctx, c); err != nil {
		return ComparisonDetail{}, err
	}
	return ComparisonDetail{Comparison: c, Reference: ref, Actual: act, Requirement: req}, nil
}

// pairAsset turns an unknown asset ID into a conflict, matching the contract's
// "409 mismatch or wrong project" for comparison inputs.
func (s *Service) pairAsset(ctx context.Context, id string) (domain.Asset, error) {
	a, err := s.repo.GetAsset(ctx, id)
	if de, ok := domain.AsError(err); ok && de.Kind == domain.KindNotFound {
		return domain.Asset{}, domain.Conflict("WRONG_PROJECT", "asset %q does not belong to this project", id)
	}
	return a, err
}

// requirementVersion reuses the latest version when the text is unchanged;
// otherwise edits create a new immutable version.
func (s *Service) requirementVersion(ctx context.Context, projectID, text string) (domain.RequirementVersion, error) {
	sha := domain.SHA256Hex([]byte(text))
	latest, err := s.repo.LatestRequirement(ctx, projectID)
	if err != nil {
		return domain.RequirementVersion{}, err
	}
	if latest != nil && latest.SHA256 == sha {
		return *latest, nil
	}
	next := 1
	if latest != nil {
		next = latest.Version + 1
	}
	r := domain.RequirementVersion{
		ID: s.ids.NewID(), ProjectID: projectID, Version: next,
		Text: text, SHA256: sha, CreatedAt: s.clock.Now(),
	}
	if err := s.repo.CreateRequirement(ctx, r); err != nil {
		return domain.RequirementVersion{}, err
	}
	return r, nil
}

func (s *Service) GetComparison(ctx context.Context, id string) (ComparisonDetail, error) {
	c, err := s.repo.GetComparison(ctx, id)
	if err != nil {
		return ComparisonDetail{}, err
	}
	ref, err := s.repo.GetAsset(ctx, c.ReferenceAssetID)
	if err != nil {
		return ComparisonDetail{}, err
	}
	act, err := s.repo.GetAsset(ctx, c.ActualAssetID)
	if err != nil {
		return ComparisonDetail{}, err
	}
	req, err := s.repo.GetRequirement(ctx, c.RequirementVersionID)
	if err != nil {
		return ComparisonDetail{}, err
	}
	runs, err := s.repo.ListRuns(ctx, id)
	if err != nil {
		return ComparisonDetail{}, err
	}
	return ComparisonDetail{Comparison: c, Reference: ref, Actual: act, Requirement: req, Runs: runs}, nil
}
