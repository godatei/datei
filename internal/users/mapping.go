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

// MapAccessTokenProjectionToAPI converts a db.UserAccountAccessTokenProjection
// to an api.PersonalAccessToken. The token hash is never exposed.
func MapAccessTokenProjectionToAPI(row *db.UserAccountAccessTokenProjection) api.PersonalAccessToken {
	return api.PersonalAccessToken{
		Id:        row.ID,
		Label:     row.Label,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}
}

// MapAccessTokenProjectionSliceToAPI converts a slice of access-token
// projections to the API shape.
func MapAccessTokenProjectionSliceToAPI(rows []db.UserAccountAccessTokenProjection) []api.PersonalAccessToken {
	tokens := make([]api.PersonalAccessToken, len(rows))
	for i := range rows {
		tokens[i] = MapAccessTokenProjectionToAPI(&rows[i])
	}
	return tokens
}

func toAdminUserListItem(row db.ListUserAccountProjectionsRow) api.AdminUserListItem {
	item := api.AdminUserListItem{
		Id:             row.ID,
		Name:           row.Name,
		IsAdmin:        row.IsAdmin,
		MfaEnabled:     row.MfaEnabled,
		Archived:       row.ArchivedAt != nil,
		CreatedAt:      row.CreatedAt,
		LastLoggedInAt: row.LastLoggedInAt,
	}
	if row.PrimaryEmail != nil {
		email := openapi_types.Email(*row.PrimaryEmail)
		item.PrimaryEmail = &email
	}
	if row.PrimaryEmailVerifiedAt != nil {
		item.PrimaryEmailVerified = new(true)
	} else if row.PrimaryEmail != nil {
		item.PrimaryEmailVerified = new(false)
	}
	return item
}
