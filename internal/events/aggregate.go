package events

import "github.com/google/uuid"

// DomainEventFor is the constraint for domain events that can apply themselves
// to a specific aggregate type.
type DomainEventFor[A any] interface {
	DomainEvent
	ApplyTo(a *A)
}

// AggregateRoot is the interface that all event-sourced aggregates satisfy
// through embedding Base and adding Replay/AggregateID wrappers.
type AggregateRoot interface {
	Replay([]DomainEvent)
	GetUncommittedEvents() []DomainEvent
	MarkEventsAsCommitted()
	Version() int
	AggregateID() uuid.UUID
}

// Base provides the event-sourcing mechanics shared by all aggregates.
// A = concrete aggregate type, E = domain event constraint.
type Base[A any, E DomainEventFor[A]] struct {
	uncommittedEvents []DomainEvent
	version           int
}

func (b *Base[A, E]) GetUncommittedEvents() []DomainEvent {
	return b.uncommittedEvents
}

func (b *Base[A, E]) MarkEventsAsCommitted() {
	b.uncommittedEvents = []DomainEvent{}
}

func (b *Base[A, E]) Version() int {
	return b.version
}

// RecordEvent appends an event, increments the version, and applies it.
// self must be a pointer to the concrete aggregate that embeds this Base.
func (b *Base[A, E]) RecordEvent(self *A, event E) {
	b.uncommittedEvents = append(b.uncommittedEvents, event)
	b.version++
	event.ApplyTo(self)
}

// ReplayEvents reconstructs aggregate state from event history and sets the version.
func (b *Base[A, E]) ReplayEvents(self *A, domainEvents []DomainEvent) {
	for _, event := range domainEvents {
		event.(E).ApplyTo(self)
	}
	b.version = len(domainEvents)
}
