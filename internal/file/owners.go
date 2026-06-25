package file

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
)

// lookupOwners resolves the owner (id + display name) for the given file IDs,
// keyed by file ID, from file_permission_projection (the authoritative source).
func lookupOwners(
	ctx context.Context, q *db.Queries, ids []uuid.UUID,
) (map[uuid.UUID]db.ListFileOwnersRow, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]db.ListFileOwnersRow{}, nil
	}
	rows, err := q.ListFileOwners(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list file owners: %w", err)
	}
	owners := make(map[uuid.UUID]db.ListFileOwnersRow, len(rows))
	for _, r := range rows {
		owners[r.FileID] = r
	}
	return owners, nil
}

// AttachOwners sets the required owner (id + name) on each file from the
// permission table. Owner is non-nullable on the API model, so every code path
// that returns api.File must call this.
//
// Every file has exactly one owner by construction: the owner permission row is
// written in the same transaction as the file projection on FileCreated (see
// updateProjectionForFileCreated). A file with no resolvable owner therefore
// signals a broken invariant, so we fail loudly rather than emit a blank owner.
func (s *Service) AttachOwners(ctx context.Context, files []api.File) error {
	if len(files) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(files))
	for i := range files {
		ids[i] = files[i].Id
	}
	owners, err := lookupOwners(ctx, db.New(s.db), ids)
	if err != nil {
		return err
	}
	for i := range files {
		o, ok := owners[files[i].Id]
		if !ok {
			return fmt.Errorf("file %s has no owner", files[i].Id)
		}
		files[i].OwnerId = o.OwnerID
		files[i].OwnerName = o.OwnerName
	}
	return nil
}

// AttachOwner sets the required owner on a single file response.
func (s *Service) AttachOwner(ctx context.Context, f *api.File) error {
	if f == nil {
		return nil
	}
	files := []api.File{*f}
	if err := s.AttachOwners(ctx, files); err != nil {
		return err
	}
	*f = files[0]
	return nil
}

// AttachTrashedOwners sets the required owner on each trashed file response.
func (s *Service) AttachTrashedOwners(ctx context.Context, files []api.TrashedFile) error {
	if len(files) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(files))
	for i := range files {
		ids[i] = files[i].Id
	}
	owners, err := lookupOwners(ctx, db.New(s.db), ids)
	if err != nil {
		return err
	}
	for i := range files {
		o, ok := owners[files[i].Id]
		if !ok {
			return fmt.Errorf("trashed file %s has no owner", files[i].Id)
		}
		files[i].OwnerId = o.OwnerID
		files[i].OwnerName = o.OwnerName
	}
	return nil
}
