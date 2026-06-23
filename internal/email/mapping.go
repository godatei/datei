package email

import (
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
)

// MapAccountProjectionToAPI converts a db.MailAccountProjection to an api.MailAccount.
// The encrypted password is intentionally never exposed.
func MapAccountProjectionToAPI(p *db.MailAccountProjection) *api.MailAccount {
	if p == nil {
		return nil
	}
	return &api.MailAccount{
		Id:        p.ID,
		Name:      p.Name,
		ImapHost:  p.ImapHost,
		ImapPort:  int(p.ImapPort),
		Username:  p.Username,
		Security:  api.MailAccountSecurity(p.Security),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// MapAccountProjectionSliceToAPI converts a slice of db.MailAccountProjection.
func MapAccountProjectionSliceToAPI(projections []db.MailAccountProjection) []api.MailAccount {
	result := make([]api.MailAccount, 0, len(projections))
	for i := range projections {
		if mapped := MapAccountProjectionToAPI(&projections[i]); mapped != nil {
			result = append(result, *mapped)
		}
	}
	return result
}

// MapRuleProjectionToAPI converts a db.MailRuleProjection to an api.MailRule.
func MapRuleProjectionToAPI(p *db.MailRuleProjection) *api.MailRule {
	if p == nil {
		return nil
	}
	return &api.MailRule{
		Id:                p.ID,
		AccountId:         p.AccountID,
		Name:              p.Name,
		Order:             int(p.SortOrder),
		Enabled:           p.Enabled,
		Folder:            p.Folder,
		FilterFrom:        p.FilterFrom,
		FilterSubject:     p.FilterSubject,
		MaxAgeDays:        int(p.MaxAgeDays),
		AttachmentPattern: p.AttachmentPattern,
		Action:            api.MailRuleAction(p.Action),
		TargetDirectoryId: p.TargetDirectoryID,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

// MapRuleProjectionSliceToAPI converts a slice of db.MailRuleProjection.
func MapRuleProjectionSliceToAPI(projections []db.MailRuleProjection) []api.MailRule {
	result := make([]api.MailRule, 0, len(projections))
	for i := range projections {
		if mapped := MapRuleProjectionToAPI(&projections[i]); mapped != nil {
			result = append(result, *mapped)
		}
	}
	return result
}
