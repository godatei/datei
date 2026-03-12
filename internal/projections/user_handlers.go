package projections

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
)

func UpdateProjectionForUserRegistered(
	ctx context.Context, q *db.Queries, event *events.UserRegisteredEvent,
) error {
	if err := q.InsertUserAccountProjection(ctx, db.InsertUserAccountProjectionParams{
		ID:           event.ID,
		Name:         event.Name,
		PasswordHash: event.PasswordHash,
		PasswordSalt: event.PasswordSalt,
		CreatedAt:    event.CreatedAt,
	}); err != nil {
		return fmt.Errorf("failed to insert user_account projection: %w", err)
	}

	if err := q.InsertUserAccountEmailProjection(ctx, db.InsertUserAccountEmailProjectionParams{
		ID:            event.EmailID,
		UserAccountID: event.ID,
		Email:         event.Email,
		IsPrimary:     true,
		CreatedAt:     event.CreatedAt,
	}); err != nil {
		return fmt.Errorf("failed to insert user_account_email projection: %w", err)
	}

	return nil
}

func UpdateProjectionForUserNameChanged(ctx context.Context, q *db.Queries, event *events.UserNameChangedEvent) error {
	return q.UpdateUserAccountProjectionName(ctx, db.UpdateUserAccountProjectionNameParams{
		Name:      event.NewName,
		UpdatedAt: event.ChangedAt,
		ID:        event.ID,
	})
}

func UpdateProjectionForUserPasswordChanged(
	ctx context.Context, q *db.Queries, event *events.UserPasswordChangedEvent,
) error {
	return q.UpdateUserAccountProjectionPassword(ctx, db.UpdateUserAccountProjectionPasswordParams{
		PasswordHash: event.PasswordHash,
		PasswordSalt: event.PasswordSalt,
		UpdatedAt:    event.ChangedAt,
		ID:           event.ID,
	})
}

func UpdateProjectionForUserEmailChanged(
	ctx context.Context, q *db.Queries, event *events.UserEmailChangedEvent,
) error {
	return q.UpdateUserAccountEmailProjectionEmail(ctx, db.UpdateUserAccountEmailProjectionEmailParams{
		Email:         event.NewEmail,
		UserAccountID: event.ID,
	})
}

func UpdateProjectionForUserEmailVerified(
	ctx context.Context, q *db.Queries, event *events.UserEmailVerifiedEvent,
) error {
	return q.UpdateUserAccountEmailProjectionVerified(ctx, db.UpdateUserAccountEmailProjectionVerifiedParams{
		VerifiedAt:    &event.VerifiedAt,
		UserAccountID: event.ID,
	})
}

func UpdateProjectionForUserEmailAdded(
	ctx context.Context, q *db.Queries, event *events.UserEmailAddedEvent,
) error {
	return q.InsertUserAccountEmailProjection(ctx, db.InsertUserAccountEmailProjectionParams{
		ID:            event.EmailID,
		UserAccountID: event.ID,
		Email:         event.Email,
		IsPrimary:     false,
		CreatedAt:     event.AddedAt,
	})
}

func UpdateProjectionForUserEmailRemoved(
	ctx context.Context, q *db.Queries, event *events.UserEmailRemovedEvent,
) error {
	return q.DeleteUserAccountEmailProjection(ctx, event.EmailID)
}

func UpdateProjectionForUserEmailSetPrimary(
	ctx context.Context, q *db.Queries, event *events.UserEmailSetPrimaryEvent,
) error {
	return q.SetUserAccountEmailPrimaryProjection(ctx, db.SetUserAccountEmailPrimaryProjectionParams{
		ID:            event.NewPrimaryEmailID,
		UserAccountID: event.ID,
	})
}

func UpdateProjectionForUserMFASetupInitiated(
	ctx context.Context, q *db.Queries, event *events.UserMFASetupInitiatedEvent,
) error {
	return q.UpdateUserAccountProjectionMFASecret(ctx, db.UpdateUserAccountProjectionMFASecretParams{
		MfaSecret: &event.MFASecret,
		UpdatedAt: event.InitiatedAt,
		ID:        event.ID,
	})
}

func UpdateProjectionForUserMFAEnabled(ctx context.Context, q *db.Queries, event *events.UserMFAEnabledEvent) error {
	if err := q.UpdateUserAccountProjectionMFAEnabled(ctx, db.UpdateUserAccountProjectionMFAEnabledParams{
		MfaEnabledAt: &event.EnabledAt,
		ID:           event.ID,
	}); err != nil {
		return fmt.Errorf("failed to enable MFA projection: %w", err)
	}

	for _, code := range event.RecoveryCodes {
		if err := q.InsertMFARecoveryCodeProjection(ctx, db.InsertMFARecoveryCodeProjectionParams{
			ID:            code.ID,
			UserAccountID: event.ID,
			CodeHash:      code.CodeHash,
			CodeSalt:      code.CodeSalt,
		}); err != nil {
			return fmt.Errorf("failed to insert recovery code projection: %w", err)
		}
	}

	return nil
}

func UpdateProjectionForUserMFADisabled(ctx context.Context, q *db.Queries, event *events.UserMFADisabledEvent) error {
	if err := q.UpdateUserAccountProjectionMFADisabled(ctx, db.UpdateUserAccountProjectionMFADisabledParams{
		UpdatedAt: event.DisabledAt,
		ID:        event.ID,
	}); err != nil {
		return err
	}
	return q.DeleteAllMFARecoveryCodesProjection(ctx, event.ID)
}

func UpdateProjectionForUserMFARecoveryCodeUsed(
	ctx context.Context, q *db.Queries, event *events.UserMFARecoveryCodeUsedEvent,
) error {
	return q.MarkMFARecoveryCodeUsedProjection(ctx, event.RecoveryCodeID)
}

func UpdateProjectionForUserMFARecoveryCodesRegenerated(
	ctx context.Context, q *db.Queries, event *events.UserMFARecoveryCodesRegeneratedEvent,
) error {
	if err := q.DeleteAllMFARecoveryCodesProjection(ctx, event.ID); err != nil {
		return err
	}

	for _, code := range event.RecoveryCodes {
		if err := q.InsertMFARecoveryCodeProjection(ctx, db.InsertMFARecoveryCodeProjectionParams{
			ID:            code.ID,
			UserAccountID: event.ID,
			CodeHash:      code.CodeHash,
			CodeSalt:      code.CodeSalt,
		}); err != nil {
			return fmt.Errorf("failed to insert recovery code projection: %w", err)
		}
	}

	return nil
}

func UpdateProjectionForUserArchived(ctx context.Context, q *db.Queries, event *events.UserArchivedEvent) error {
	return q.UpdateUserAccountProjectionArchived(ctx, db.UpdateUserAccountProjectionArchivedParams{
		ArchivedAt: &event.ArchivedAt,
		ID:         event.ID,
	})
}

func UpdateProjectionForUserLoggedIn(ctx context.Context, q *db.Queries, event *events.UserLoggedInEvent) error {
	return q.UpdateUserAccountProjectionLoggedIn(ctx, db.UpdateUserAccountProjectionLoggedInParams{
		LastLoggedInAt: &event.LoggedInAt,
		ID:             event.ID,
	})
}
