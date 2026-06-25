package email

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
)

func updateProjectionForMailAccountCreated(ctx context.Context, q *db.Queries, event *MailAccountCreatedEvent) error {
	if err := q.InsertMailAccountProjection(ctx, db.InsertMailAccountProjectionParams{
		ID:                event.ID,
		OwnerID:           event.OwnerID,
		Name:              event.Name,
		ImapHost:          event.ImapHost,
		ImapPort:          int32(event.ImapPort),
		Username:          event.Username,
		PasswordEncrypted: event.PasswordEncrypted,
		Security:          db.MailSecurity(event.Security),
		CreatedAt:         event.CreatedAt,
		UpdatedAt:         event.CreatedAt,
	}); err != nil {
		return fmt.Errorf("failed to insert mail_account_projection: %w", err)
	}
	return nil
}

func updateProjectionForMailAccountUpdated(ctx context.Context, q *db.Queries, event *MailAccountUpdatedEvent) error {
	if err := q.UpdateMailAccountProjection(ctx, db.UpdateMailAccountProjectionParams{
		Name:              event.Name,
		ImapHost:          event.ImapHost,
		ImapPort:          int32(event.ImapPort),
		Username:          event.Username,
		PasswordEncrypted: event.PasswordEncrypted,
		Security:          db.MailSecurity(event.Security),
		UpdatedAt:         event.UpdatedAt,
		ID:                event.ID,
	}); err != nil {
		return fmt.Errorf("failed to update mail_account_projection: %w", err)
	}
	return nil
}

func updateProjectionForMailAccountDeleted(ctx context.Context, q *db.Queries, event *MailAccountDeletedEvent) error {
	if err := q.DeleteMailAccountProjection(ctx, event.ID); err != nil {
		return fmt.Errorf("failed to delete mail_account_projection: %w", err)
	}
	return nil
}

func updateProjectionForMailRuleCreated(ctx context.Context, q *db.Queries, event *MailRuleCreatedEvent) error {
	if err := q.InsertMailRuleProjection(ctx, db.InsertMailRuleProjectionParams{
		ID:                event.ID,
		AccountID:         event.AccountID,
		Name:              event.Name,
		SortOrder:         int32(event.Order),
		Enabled:           event.Enabled,
		Folder:            event.Folder,
		FilterFrom:        event.FilterFrom,
		FilterSubject:     event.FilterSubject,
		MaxAgeDays:        int32(event.MaxAgeDays),
		AttachmentPattern: event.AttachmentPattern,
		Action:            dbMailAction(event.Action),
		TargetDirectoryID: event.TargetDirectoryID,
		CreatedAt:         event.CreatedAt,
		UpdatedAt:         event.CreatedAt,
	}); err != nil {
		return fmt.Errorf("failed to insert mail_rule_projection: %w", err)
	}
	return nil
}

func updateProjectionForMailRuleUpdated(ctx context.Context, q *db.Queries, event *MailRuleUpdatedEvent) error {
	if err := q.UpdateMailRuleProjection(ctx, db.UpdateMailRuleProjectionParams{
		AccountID:         event.AccountID,
		Name:              event.Name,
		SortOrder:         int32(event.Order),
		Enabled:           event.Enabled,
		Folder:            event.Folder,
		FilterFrom:        event.FilterFrom,
		FilterSubject:     event.FilterSubject,
		MaxAgeDays:        int32(event.MaxAgeDays),
		AttachmentPattern: event.AttachmentPattern,
		Action:            dbMailAction(event.Action),
		TargetDirectoryID: event.TargetDirectoryID,
		UpdatedAt:         event.UpdatedAt,
		ID:                event.ID,
	}); err != nil {
		return fmt.Errorf("failed to update mail_rule_projection: %w", err)
	}
	return nil
}

// dbMailAction converts a domain action into the nullable projection enum; a nil
// action (no action) maps to a NULL column.
func dbMailAction(a *Action) *db.MailAction {
	if a == nil {
		return nil
	}
	v := db.MailAction(*a)
	return &v
}

func updateProjectionForMailRuleDeleted(ctx context.Context, q *db.Queries, event *MailRuleDeletedEvent) error {
	if err := q.DeleteMailRuleProjection(ctx, event.ID); err != nil {
		return fmt.Errorf("failed to delete mail_rule_projection: %w", err)
	}
	return nil
}
