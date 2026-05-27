package link

import (
	"context"
	"time"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LinkEvent extends DomainEvent with the ability to apply itself to an Aggregate.
type LinkEvent interface {
	events.DomainEvent
	ApplyTo(a *Aggregate)
}

// NewEventStore creates an event store for the link_event table.
//
//nolint:dupl // each domain wires its own sqlc queries into the generic store
func NewEventStore(pool *pgxpool.Pool) *events.PostgresEventStore {
	return events.NewStore(pool, events.StoreQueries{
		GetVersion: func(ctx context.Context, q *db.Queries, id uuid.UUID) (int32, error) {
			return q.GetLinkStreamVersion(ctx, id)
		},
		Insert: func(ctx context.Context, q *db.Queries, p events.InsertParams) error {
			return q.InsertLinkEvent(ctx, db.InsertLinkEventParams{
				StreamID: p.StreamID, StreamVersion: p.StreamVersion,
				EventType: p.EventType, EventData: p.EventData,
			})
		},
		GetEvents: func(ctx context.Context, q *db.Queries, id uuid.UUID, from int32) ([]events.EventRow, error) {
			rows, err := q.GetLinkEventsByStreamID(ctx, db.GetLinkEventsByStreamIDParams{
				StreamID: id, StreamVersion: from,
			})
			if err != nil {
				return nil, err
			}
			out := make([]events.EventRow, len(rows))
			for i, r := range rows {
				out[i] = events.EventRow{EventType: r.EventType, EventData: r.EventData}
			}
			return out, nil
		},
	})
}

func init() {
	events.RegisterEvent("LinkCreated", func() events.DomainEvent { return &LinkCreatedEvent{} })
	events.RegisterEvent("LinkUpdated", func() events.DomainEvent { return &LinkUpdatedEvent{} })
	events.RegisterEvent("LinkKeyRotated", func() events.DomainEvent { return &LinkKeyRotatedEvent{} })
	events.RegisterEvent("LinkDateiAdded", func() events.DomainEvent { return &LinkDateiAddedEvent{} })
	events.RegisterEvent("LinkDateiRemoved", func() events.DomainEvent { return &LinkDateiRemovedEvent{} })
	events.RegisterEvent("LinkRevoked", func() events.DomainEvent { return &LinkRevokedEvent{} })
	events.RegisterEvent("LinkOpened", func() events.DomainEvent { return &LinkOpenedEvent{} })
}

// ============================================================================
// Link Events
// ============================================================================

type LinkCreatedEvent struct {
	ID        uuid.UUID   `json:"id"`
	OwnerID   uuid.UUID   `json:"owner_id"`
	Name      string      `json:"name"`
	Key       string      `json:"key"`
	Code      *string     `json:"code,omitempty"`
	ExpiresAt *time.Time  `json:"expires_at,omitempty"`
	DateiIDs  []uuid.UUID `json:"datei_ids"`
	CreatedAt time.Time   `json:"created_at"`
}

func (e LinkCreatedEvent) EventType() string   { return "LinkCreated" }
func (e LinkCreatedEvent) StreamID() uuid.UUID { return e.ID }
func (e LinkCreatedEvent) ApplyTo(a *Aggregate) {
	a.ID = e.ID
	a.OwnerID = e.OwnerID
	a.Name = e.Name
	a.Key = e.Key
	a.Code = e.Code
	a.ExpiresAt = e.ExpiresAt
	a.CreatedAt = e.CreatedAt
	a.UpdatedAt = e.CreatedAt
	if a.dateiIDs == nil {
		a.dateiIDs = make(map[uuid.UUID]struct{}, len(e.DateiIDs))
	}
	for _, id := range e.DateiIDs {
		a.dateiIDs[id] = struct{}{}
	}
}

// LinkUpdatedEvent is recorded when the owner edits a link's settings (name,
// code, expiration). These are batched in a single event because they map to
// a single Save action in the edit modal — a deliberate deviation from the
// per-field event pattern used in the datei/users domains.
type LinkUpdatedEvent struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Code      *string    `json:"code,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (e LinkUpdatedEvent) EventType() string   { return "LinkUpdated" }
func (e LinkUpdatedEvent) StreamID() uuid.UUID { return e.ID }
func (e LinkUpdatedEvent) ApplyTo(a *Aggregate) {
	a.Name = e.Name
	a.Code = e.Code
	a.ExpiresAt = e.ExpiresAt
	a.UpdatedAt = e.UpdatedAt
}

type LinkKeyRotatedEvent struct {
	ID        uuid.UUID `json:"id"`
	OldKey    string    `json:"old_key"`
	NewKey    string    `json:"new_key"`
	RotatedAt time.Time `json:"rotated_at"`
}

func (e LinkKeyRotatedEvent) EventType() string   { return "LinkKeyRotated" }
func (e LinkKeyRotatedEvent) StreamID() uuid.UUID { return e.ID }
func (e LinkKeyRotatedEvent) ApplyTo(a *Aggregate) {
	a.Key = e.NewKey
	a.UpdatedAt = e.RotatedAt
}

type LinkDateiAddedEvent struct {
	ID      uuid.UUID `json:"id"`
	DateiID uuid.UUID `json:"datei_id"`
	AddedAt time.Time `json:"added_at"`
}

func (e LinkDateiAddedEvent) EventType() string   { return "LinkDateiAdded" }
func (e LinkDateiAddedEvent) StreamID() uuid.UUID { return e.ID }
func (e LinkDateiAddedEvent) ApplyTo(a *Aggregate) {
	if a.dateiIDs == nil {
		a.dateiIDs = make(map[uuid.UUID]struct{})
	}
	a.dateiIDs[e.DateiID] = struct{}{}
	a.UpdatedAt = e.AddedAt
}

type LinkDateiRemovedEvent struct {
	ID        uuid.UUID `json:"id"`
	DateiID   uuid.UUID `json:"datei_id"`
	RemovedAt time.Time `json:"removed_at"`
}

func (e LinkDateiRemovedEvent) EventType() string   { return "LinkDateiRemoved" }
func (e LinkDateiRemovedEvent) StreamID() uuid.UUID { return e.ID }
func (e LinkDateiRemovedEvent) ApplyTo(a *Aggregate) {
	delete(a.dateiIDs, e.DateiID)
	a.UpdatedAt = e.RemovedAt
}

type LinkOpenedEvent struct {
	ID       uuid.UUID `json:"id"`
	OpenedAt time.Time `json:"opened_at"`
}

func (e LinkOpenedEvent) EventType() string   { return "LinkOpened" }
func (e LinkOpenedEvent) StreamID() uuid.UUID { return e.ID }
func (e LinkOpenedEvent) ApplyTo(a *Aggregate) {
	a.UpdatedAt = e.OpenedAt
}

type LinkRevokedEvent struct {
	ID        uuid.UUID `json:"id"`
	RevokedAt time.Time `json:"revoked_at"`
}

func (e LinkRevokedEvent) EventType() string   { return "LinkRevoked" }
func (e LinkRevokedEvent) StreamID() uuid.UUID { return e.ID }
func (e LinkRevokedEvent) ApplyTo(a *Aggregate) {
	revokedAt := e.RevokedAt
	a.RevokedAt = &revokedAt
	a.UpdatedAt = e.RevokedAt
}
