package mapping

import (
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
)

// MapDBVersionToAPI converts a database DateiVersion to an API DateiVersion
func MapDBVersionToAPI(dbVersion *db.DateiVersion) *api.DateiVersion {
	if dbVersion == nil {
		return nil
	}

	apiVersion := &api.DateiVersion{
		Checksum:  dbVersion.Checksum,
		CreatedAt: dbVersion.CreatedAt,
		FileSize:  dbVersion.FileSize,
		Id:        dbVersion.ID,
		MimeType:  dbVersion.MimeType,
	}

	if dbVersion.ContentMd != nil {
		apiVersion.ContentMd = dbVersion.ContentMd
	}

	if dbVersion.CreatedBy != nil {
		createdByUUID := *dbVersion.CreatedBy
		apiVersion.CreatedBy = &createdByUUID
	}

	return apiVersion
}

// MapDBDateiToAPI converts a database Datei to an API DateiResponse
func MapDBDateiToAPI(dbDatei *db.Datei, latestVersion *db.DateiVersion, name *string) *api.DateiResponse {
	if dbDatei == nil {
		return nil
	}

	response := &api.DateiResponse{
		CreatedAt:   dbDatei.CreatedAt,
		Id:          dbDatei.ID,
		IsDirectory: dbDatei.IsDirectory,
		Name:        name,
		UpdatedAt:   dbDatei.UpdatedAt,
	}

	// Map optional parent ID
	if dbDatei.ParentID != nil {
		response.ParentId = dbDatei.ParentID
	}

	// Map optional linked Datei ID
	if dbDatei.LinkedDateiID != nil {
		response.LinkedDateiId = dbDatei.LinkedDateiID
	}

	// Map optional created by
	if dbDatei.CreatedBy != nil {
		response.CreatedBy = dbDatei.CreatedBy
	}

	// Map trashed info
	if dbDatei.TrashedAt != nil {
		response.TrashedAt = dbDatei.TrashedAt
	}

	if dbDatei.TrashedBy != nil {
		response.TrashedBy = dbDatei.TrashedBy
	}

	// Map latest version if provided
	if latestVersion != nil {
		response.LatestVersion = MapDBVersionToAPI(latestVersion)
	}

	return response
}

// MapDBDateiSliceToAPI converts a slice of database Datei to API DateiResponse slice
func MapDBDateiSliceToAPI(
	dbDateiList []db.Datei,
	versions map[uuid.UUID]*db.DateiVersion,
	names map[uuid.UUID]*string,
) []api.DateiResponse {
	result := make([]api.DateiResponse, 0, len(dbDateiList))

	for _, dbDatei := range dbDateiList {
		var latestVersion *db.DateiVersion
		var name *string

		// Get version if available
		if dbDatei.LatestVersionID != nil {
			latestVersion = versions[*dbDatei.LatestVersionID]
		}

		// Get name if available
		if dbDatei.LatestNameID != nil {
			name = names[*dbDatei.LatestNameID]
		}

		// Create a copy for safety
		dateiCopy := dbDatei
		mapped := MapDBDateiToAPI(&dateiCopy, latestVersion, name)
		if mapped != nil {
			result = append(result, *mapped)
		}
	}

	return result
}
