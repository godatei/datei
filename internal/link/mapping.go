package link

import (
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
)

// MapProjectionToLink converts a db.LinkProjection plus computed counts into
// the list-shape api.Link (no dateien).
func MapProjectionToLink(p *db.LinkProjection, fileCount, folderCount int) *api.Link {
	if p == nil {
		return nil
	}
	return &api.Link{
		Id:          p.ID,
		OwnerId:     p.OwnerID,
		Name:        p.Name,
		AccessToken: p.AccessToken,
		Code:        p.Code,
		ExpiresAt:   p.ExpiresAt,
		RevokedAt:   p.RevokedAt,
		FileCount:   fileCount,
		FolderCount: folderCount,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// MapAggregateToLinkDetail converts an in-memory Aggregate plus its top-level
// dateien and computed counts into the detail-shape api.LinkDetail, used after
// a Save to avoid refetching the link_projection row we just wrote.
func MapAggregateToLinkDetail(a *Aggregate, dateien []db.DateiProjection, fileCount, folderCount int) *api.LinkDetail {
	if a == nil {
		return nil
	}
	return &api.LinkDetail{
		Id:          a.ID,
		OwnerId:     a.OwnerID,
		Name:        a.Name,
		AccessToken: a.AccessToken,
		Code:        a.Code,
		ExpiresAt:   a.ExpiresAt,
		RevokedAt:   a.RevokedAt,
		Dateien:     datei.MapProjectionSliceToAPI(dateien),
		FileCount:   fileCount,
		FolderCount: folderCount,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}
