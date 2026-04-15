package users

import (
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// MapEmailProjectionToAPI converts a db.UserAccountEmailProjection to an api.UserEmail.
func MapEmailProjectionToAPI(row *db.UserAccountEmailProjection) api.UserEmail {
	return api.UserEmail{
		Id:        row.ID,
		Email:     openapi_types.Email(row.Email),
		IsPrimary: row.IsPrimary,
		Verified:  row.VerifiedAt != nil,
		CreatedAt: row.CreatedAt,
	}
}

// MapEmailProjectionSliceToAPI converts a slice of db.UserAccountEmailProjection to a slice of api.UserEmail.
func MapEmailProjectionSliceToAPI(rows []db.UserAccountEmailProjection) []api.UserEmail {
	emails := make([]api.UserEmail, len(rows))
	for i := range rows {
		emails[i] = MapEmailProjectionToAPI(&rows[i])
	}
	return emails
}
