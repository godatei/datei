package datei

import (
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
)

// MapProjectionToAPI converts a db.DateiProjection to an api.Datei.
func MapProjectionToAPI(p *db.DateiProjection) *api.Datei {
	if p == nil {
		return nil
	}

	result := &api.Datei{
		Id:          p.ID,
		IsDirectory: p.IsDirectory,
		Name:        &p.Name,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Size:        p.Size,
		Checksum:    p.Checksum,
		MimeType:    p.MimeType,
		ContentMd:   p.ContentMd,
	}

	if p.ParentID != nil {
		result.ParentId = p.ParentID
	}
	if p.LinkedDateiID != nil {
		result.LinkedDateiId = p.LinkedDateiID
	}
	if p.CreatedBy != nil {
		result.CreatedBy = p.CreatedBy
	}
	if p.TrashedAt != nil {
		result.TrashedAt = p.TrashedAt
	}
	if p.TrashedBy != nil {
		result.TrashedBy = p.TrashedBy
	}
	if p.UpdatedBy != nil {
		result.UpdatedBy = p.UpdatedBy
	}

	return result
}

// MapProjectionSliceToAPI converts a slice of db.DateiProjection to a slice of api.Datei.
func MapProjectionSliceToAPI(projections []db.DateiProjection) []api.Datei {
	result := make([]api.Datei, 0, len(projections))
	for i := range projections {
		if mapped := MapProjectionToAPI(&projections[i]); mapped != nil {
			result = append(result, *mapped)
		}
	}
	return result
}

// MapProjectionToTrashedDatei converts a db.DateiProjection to an api.TrashedDatei,
// with an optional originPath for root-level trash items.
func MapProjectionToTrashedDatei(p *db.DateiProjection, originPath *[]api.DateiPathItem) *api.TrashedDatei {
	if p == nil {
		return nil
	}

	result := &api.TrashedDatei{
		Id:          p.ID,
		IsDirectory: p.IsDirectory,
		Name:        &p.Name,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Size:        p.Size,
		Checksum:    p.Checksum,
		MimeType:    p.MimeType,
		ContentMd:   p.ContentMd,
		TrashedAt:   p.TrashedAt,
		OriginPath:  originPath,
	}

	if p.ParentID != nil {
		result.ParentId = p.ParentID
	}
	if p.LinkedDateiID != nil {
		result.LinkedDateiId = p.LinkedDateiID
	}
	if p.CreatedBy != nil {
		result.CreatedBy = p.CreatedBy
	}
	if p.TrashedBy != nil {
		result.TrashedBy = p.TrashedBy
	}
	if p.UpdatedBy != nil {
		result.UpdatedBy = p.UpdatedBy
	}

	return result
}

// MapAggregateToAPI converts an Aggregate to an api.Datei.
func MapAggregateToAPI(a *Aggregate) *api.Datei {
	if a == nil {
		return nil
	}

	result := &api.Datei{
		Id:          a.ID,
		IsDirectory: a.IsDirectory,
		Name:        &a.Name,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		Size:        a.Size,
		Checksum:    a.Checksum,
		MimeType:    a.MimeType,
		ContentMd:   a.ContentMD,
	}

	if a.ParentID != nil {
		result.ParentId = a.ParentID
	}
	if a.LinkedDateiID != nil {
		result.LinkedDateiId = a.LinkedDateiID
	}
	if a.CreatedBy != uuid.Nil {
		result.CreatedBy = &a.CreatedBy
	}
	if a.TrashedAt != nil {
		result.TrashedAt = a.TrashedAt
	}
	if a.TrashedBy != nil {
		result.TrashedBy = a.TrashedBy
	}
	if a.UpdatedBy != uuid.Nil {
		result.UpdatedBy = &a.UpdatedBy
	}

	return result
}
