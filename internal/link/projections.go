package link

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
)

func updateProjectionForLinkCreated(ctx context.Context, q *db.Queries, event *LinkCreatedEvent) error {
	if err := q.InsertLinkProjection(ctx, db.InsertLinkProjectionParams{
		ID:          event.ID,
		OwnerID:     event.OwnerID,
		Name:        event.Name,
		AccessToken: event.AccessToken,
		Code:        event.Code,
		ExpiresAt:   event.ExpiresAt,
		CreatedAt:   event.CreatedAt,
	}); err != nil {
		return fmt.Errorf("failed to insert link_projection: %w", err)
	}

	for _, dateiID := range event.DateiIDs {
		if err := q.InsertLinkDateiProjection(ctx, db.InsertLinkDateiProjectionParams{
			LinkID:  event.ID,
			DateiID: dateiID,
			AddedAt: event.CreatedAt,
		}); err != nil {
			return fmt.Errorf("failed to insert link_datei_projection: %w", err)
		}
	}

	return nil
}

func updateProjectionForLinkUpdated(ctx context.Context, q *db.Queries, event *LinkUpdatedEvent) error {
	if err := q.UpdateLinkProjection(ctx, db.UpdateLinkProjectionParams{
		Name:      event.Name,
		Code:      event.Code,
		ExpiresAt: event.ExpiresAt,
		UpdatedAt: event.UpdatedAt,
		ID:        event.ID,
	}); err != nil {
		return fmt.Errorf("failed to update link_projection: %w", err)
	}
	return nil
}

func updateProjectionForLinkAccessTokenRotated(
	ctx context.Context, q *db.Queries, event *LinkAccessTokenRotatedEvent,
) error {
	if err := q.UpdateLinkProjectionAccessToken(ctx, db.UpdateLinkProjectionAccessTokenParams{
		AccessToken: event.NewAccessToken,
		UpdatedAt:   event.RotatedAt,
		ID:          event.ID,
	}); err != nil {
		return fmt.Errorf("failed to update link_projection access_token: %w", err)
	}
	return nil
}

func updateProjectionForLinkDateiAdded(ctx context.Context, q *db.Queries, event *LinkDateiAddedEvent) error {
	if err := q.InsertLinkDateiProjection(ctx, db.InsertLinkDateiProjectionParams{
		LinkID:  event.ID,
		DateiID: event.DateiID,
		AddedAt: event.AddedAt,
	}); err != nil {
		return fmt.Errorf("failed to insert link_datei_projection: %w", err)
	}
	return nil
}

func updateProjectionForLinkDateiRemoved(ctx context.Context, q *db.Queries, event *LinkDateiRemovedEvent) error {
	if err := q.DeleteLinkDateiProjection(ctx, db.DeleteLinkDateiProjectionParams{
		LinkID:  event.ID,
		DateiID: event.DateiID,
	}); err != nil {
		return fmt.Errorf("failed to delete link_datei_projection: %w", err)
	}
	return nil
}

func updateProjectionForLinkRevoked(ctx context.Context, q *db.Queries, event *LinkRevokedEvent) error {
	if err := q.UpdateLinkProjectionRevoked(ctx, db.UpdateLinkProjectionRevokedParams{
		RevokedAt: &event.RevokedAt,
		ID:        event.ID,
	}); err != nil {
		return fmt.Errorf("failed to update link_projection revoked: %w", err)
	}
	return nil
}
