package users

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
)

func updateProjectionForUserRegistered(ctx context.Context, q *db.Queries, event *UserRegisteredEvent) error {
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

func updateProjectionForUserNameChanged(ctx context.Context, q *db.Queries, event *UserNameChangedEvent) error {
	return q.UpdateUserAccountProjectionName(ctx, db.UpdateUserAccountProjectionNameParams{
		Name:      event.NewName,
		UpdatedAt: event.ChangedAt,
		ID:        event.ID,
	})
}

func updateProjectionForUserPasswordChanged(ctx context.Context, q *db.Queries, event *UserPasswordChangedEvent) error {
	return q.UpdateUserAccountProjectionPassword(ctx, db.UpdateUserAccountProjectionPasswordParams{
		PasswordHash: event.PasswordHash,
		PasswordSalt: event.PasswordSalt,
		UpdatedAt:    event.ChangedAt,
		ID:           event.ID,
	})
}

func updateProjectionForUserEmailChanged(ctx context.Context, q *db.Queries, event *UserEmailChangedEvent) error {
	return q.UpdateUserAccountEmailProjectionEmail(ctx, db.UpdateUserAccountEmailProjectionEmailParams{
		Email:         event.NewEmail,
		UserAccountID: event.ID,
	})
}

func updateProjectionForUserEmailVerified(ctx context.Context, q *db.Queries, event *UserEmailVerifiedEvent) error {
	return q.UpdateUserAccountEmailProjectionVerified(ctx, db.UpdateUserAccountEmailProjectionVerifiedParams{
		VerifiedAt:    &event.VerifiedAt,
		UserAccountID: event.ID,
	})
}

func updateProjectionForUserEmailAdded(ctx context.Context, q *db.Queries, event *UserEmailAddedEvent) error {
	return q.InsertUserAccountEmailProjection(ctx, db.InsertUserAccountEmailProjectionParams{
		ID:            event.EmailID,
		UserAccountID: event.ID,
		Email:         event.Email,
		IsPrimary:     false,
		CreatedAt:     event.AddedAt,
	})
}

func updateProjectionForUserEmailRemoved(ctx context.Context, q *db.Queries, event *UserEmailRemovedEvent) error {
	return q.DeleteUserAccountEmailProjection(ctx, event.EmailID)
}

func updateProjectionForUserEmailSetPrimary(ctx context.Context, q *db.Queries, event *UserEmailSetPrimaryEvent) error {
	return q.SetUserAccountEmailPrimaryProjection(ctx, db.SetUserAccountEmailPrimaryProjectionParams{
		ID:            event.NewPrimaryEmailID,
		UserAccountID: event.ID,
	})
}

func updateProjectionForUserMFASetupInitiated(
	ctx context.Context, q *db.Queries, event *UserMFASetupInitiatedEvent,
) error {
	return q.UpdateUserAccountProjectionMFASecret(ctx, db.UpdateUserAccountProjectionMFASecretParams{
		MfaSecret: &event.MFASecret,
		UpdatedAt: event.InitiatedAt,
		ID:        event.ID,
	})
}

func updateProjectionForUserMFAEnabled(ctx context.Context, q *db.Queries, event *UserMFAEnabledEvent) error {
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

func updateProjectionForUserMFADisabled(ctx context.Context, q *db.Queries, event *UserMFADisabledEvent) error {
	if err := q.UpdateUserAccountProjectionMFADisabled(ctx, db.UpdateUserAccountProjectionMFADisabledParams{
		UpdatedAt: event.DisabledAt,
		ID:        event.ID,
	}); err != nil {
		return err
	}
	return q.DeleteAllMFARecoveryCodesProjection(ctx, event.ID)
}

func updateProjectionForUserMFARecoveryCodeUsed(
	ctx context.Context, q *db.Queries, event *UserMFARecoveryCodeUsedEvent,
) error {
	return q.MarkMFARecoveryCodeUsedProjection(ctx, event.RecoveryCodeID)
}

func updateProjectionForUserMFARecoveryCodesRegenerated(
	ctx context.Context, q *db.Queries, event *UserMFARecoveryCodesRegeneratedEvent,
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

func updateProjectionForUserArchived(ctx context.Context, q *db.Queries, event *UserArchivedEvent) error {
	return q.UpdateUserAccountProjectionArchived(ctx, db.UpdateUserAccountProjectionArchivedParams{
		ArchivedAt: &event.ArchivedAt,
		ID:         event.ID,
	})
}

func updateProjectionForUserLoggedIn(ctx context.Context, q *db.Queries, event *UserLoggedInEvent) error {
	return q.UpdateUserAccountProjectionLoggedIn(ctx, db.UpdateUserAccountProjectionLoggedInParams{
		LastLoggedInAt: &event.LoggedInAt,
		ID:             event.ID,
	})
}
