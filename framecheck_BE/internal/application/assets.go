package application

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg" // register decoder
	_ "image/png"  // register decoder
	"net/http"

	"framecheck/internal/domain"
)

type IngestAssetInput struct {
	ProjectID string
	Role      domain.ImageRole
	Capture   domain.Capture
	Data      []byte
}

var allowedMIME = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpeg",
}

// IngestAsset validates an uploaded image from its bytes (never the client's
// filename or Content-Type), stores it by hash, and records its metadata.
func (s *Service) IngestAsset(ctx context.Context, in IngestAssetInput) (domain.Asset, error) {
	if _, err := s.repo.GetProject(ctx, in.ProjectID); err != nil {
		return domain.Asset{}, err
	}
	if !in.Role.Valid() {
		return domain.Asset{}, domain.Invalid("INVALID_ROLE", "role must be reference or actual")
	}
	if err := in.Capture.Validate(); err != nil {
		return domain.Asset{}, err
	}
	if len(in.Data) == 0 {
		return domain.Asset{}, domain.Invalid("INVALID_IMAGE", "file is empty")
	}
	if len(in.Data) > domain.MaxImageBytes {
		return domain.Asset{}, domain.TooLarge("IMAGE_TOO_LARGE", "image must be at most %d MB", domain.MaxImageBytes>>20)
	}

	mime := http.DetectContentType(in.Data)
	format, ok := allowedMIME[mime]
	if !ok {
		return domain.Asset{}, domain.Invalid("INVALID_IMAGE", "only PNG and JPEG images are accepted")
	}
	cfg, decodedFormat, err := image.DecodeConfig(bytes.NewReader(in.Data))
	if err != nil || decodedFormat != format {
		return domain.Asset{}, domain.Invalid("INVALID_IMAGE", "image could not be decoded")
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > domain.MaxImageSide || cfg.Height > domain.MaxImageSide {
		return domain.Asset{}, domain.Invalid("INVALID_IMAGE", "image sides must be 1-%d pixels", domain.MaxImageSide)
	}
	// Full decode catches truncated or corrupt pixel data that headers hide.
	if _, _, err := image.Decode(bytes.NewReader(in.Data)); err != nil {
		return domain.Asset{}, domain.Invalid("INVALID_IMAGE", "image pixel data is corrupt")
	}

	sha := domain.SHA256Hex(in.Data)
	ext := map[string]string{"png": ".png", "jpeg": ".jpg"}[format]
	key := "assets/" + sha[:2] + "/" + sha + ext
	if err := s.store.Put(ctx, key, in.Data); err != nil {
		return domain.Asset{}, err
	}

	a := domain.Asset{
		ID:         s.ids.NewID(),
		ProjectID:  in.ProjectID,
		Role:       in.Role,
		SHA256:     sha,
		MIME:       mime,
		Bytes:      int64(len(in.Data)),
		WidthPx:    cfg.Width,
		HeightPx:   cfg.Height,
		Capture:    in.Capture,
		StorageKey: key,
		CreatedAt:  s.clock.Now(),
	}
	if err := s.repo.CreateAsset(ctx, a); err != nil {
		return domain.Asset{}, err
	}
	return a, nil
}
