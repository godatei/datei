package mapping

import (
	"github.com/godatei/datei/internal/aggregate"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
)

// MapProjectionVersionToAPI converts the embedded version fields of a DateiProjection to an API DateiVersion.
// Returns nil when no version has been uploaded yet.
func MapProjectionVersionToAPI(p *db.DateiProjection) *api.DateiVersion {
	if p.LatestVersionS3Key == nil {
		return nil
	}

	v := &api.DateiVersion{
		Checksum: *p.LatestVersionChecksum,
		FileSize: *p.LatestVersionFileSize,
		MimeType: *p.LatestVersionMimeType,
	}

	if p.LatestVersionContentMd != nil {
		v.ContentMd = p.LatestVersionContentMd
	}

	return v
}

// MapDateiProjectionToAPI converts a db.DateiProjection to an api.Datei.
func MapDateiProjectionToAPI(p *db.DateiProjection) *api.Datei {
	if p == nil {
		return nil
	}

	result := &api.Datei{
		Id:            p.ID,
		IsDirectory:   p.IsDirectory,
		Name:          &p.LatestName,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		LatestVersion: MapProjectionVersionToAPI(p),
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

	return result
}

// MapDateiProjectionSliceToAPI converts a slice of db.DateiProjection to a slice of api.Datei.
func MapDateiProjectionSliceToAPI(projections []db.DateiProjection) []api.Datei {
	result := make([]api.Datei, 0, len(projections))
	for i := range projections {
		if mapped := MapDateiProjectionToAPI(&projections[i]); mapped != nil {
			result = append(result, *mapped)
		}
	}
	return result
}

// MapAggregateToAPI converts a DateiAggregate to an api.Datei.
func MapAggregateToAPI(a *aggregate.DateiAggregate) *api.Datei {
	if a == nil {
		return nil
	}

	result := &api.Datei{
		Id:          a.ID,
		IsDirectory: a.IsDirectory,
		Name:        &a.CurrentName,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
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

	if a.CurrentVersion != nil {
		result.LatestVersion = &api.DateiVersion{
			Checksum:  a.CurrentVersion.Checksum,
			FileSize:  a.CurrentVersion.FileSize,
			MimeType:  a.CurrentVersion.MimeType,
			ContentMd: a.CurrentVersion.ContentMD,
		}
	}

	return result
}
