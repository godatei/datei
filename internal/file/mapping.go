package file

import (
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
)

// MapProjectionToAPI converts a db.FileProjection to an api.File.
func MapProjectionToAPI(p *db.FileProjection) *api.File {
	if p == nil {
		return nil
	}

	result := &api.File{
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
	if p.LinkedFileID != nil {
		result.LinkedFileId = p.LinkedFileID
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

// MapProjectionSliceToAPI converts a slice of db.FileProjection to a slice of api.File.
func MapProjectionSliceToAPI(projections []db.FileProjection) []api.File {
	result := make([]api.File, 0, len(projections))
	for i := range projections {
		if mapped := MapProjectionToAPI(&projections[i]); mapped != nil {
			result = append(result, *mapped)
		}
	}
	return result
}

// MapProjectionToTrashedFile converts a db.FileProjection to an api.TrashedFile,
// with an optional originPath for root-level trash items.
func MapProjectionToTrashedFile(p *db.FileProjection, originPath *[]api.FilePathItem) *api.TrashedFile {
	if p == nil {
		return nil
	}

	result := &api.TrashedFile{
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
	if p.LinkedFileID != nil {
		result.LinkedFileId = p.LinkedFileID
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

// MapAggregateToAPI converts an Aggregate to an api.File.
func MapAggregateToAPI(a *Aggregate) *api.File {
	if a == nil {
		return nil
	}

	result := &api.File{
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
	if a.LinkedFileID != nil {
		result.LinkedFileId = a.LinkedFileID
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
