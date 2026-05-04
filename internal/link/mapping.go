package link

import (
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
)

// MapProjectionToAPI converts a db.LinkProjection plus its top-level dateien
// and computed counts into an api.Link.
func MapProjectionToAPI(p *db.LinkProjection, dateien []db.DateiProjection, fileCount, folderCount int) *api.Link {
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
		Dateien:     datei.MapProjectionSliceToAPI(dateien),
		FileCount:   fileCount,
		FolderCount: folderCount,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
