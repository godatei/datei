package mapping

import (
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
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

// MapDBDateiDetailsSliceToAPI converts a slice of ListDateiWithDetailsRow to API DateiResponse slice
func MapDBDateiDetailsSliceToAPI(details []db.ListDateiWithDetailsRow) []api.DateiResponse {
	result := make([]api.DateiResponse, 0, len(details))
	for _, row := range details {
		mapped := MapDBDateiToAPI(&row.Datei, &row.DateiVersion, &row.DateiName.Name)
		if mapped != nil {
			result = append(result, *mapped)
		}
	}
	return result
}
