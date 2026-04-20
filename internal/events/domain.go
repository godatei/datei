package events

import "github.com/google/uuid"

// DomainEvent is the base interface for all domain events
type DomainEvent interface {
	EventType() string
	StreamID() uuid.UUID
}
