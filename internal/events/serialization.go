package events

import (
	"encoding/json"
	"fmt"
)

var eventFactories = map[string]func() DomainEvent{}

// RegisterEvent adds an event type to the deserialization registry.
// Call this from init() in each domain's event file.
func RegisterEvent(eventType string, factory func() DomainEvent) {
	eventFactories[eventType] = factory
}

// Deserialize unmarshals event data from JSON using the event type
func Deserialize(eventType string, data []byte) (DomainEvent, error) {
	factory, ok := eventFactories[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	ptr := factory()
	if err := json.Unmarshal(data, ptr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", eventType, err)
	}

	return ptr, nil
}

// Serialize marshals an event to JSON
func Serialize(event DomainEvent) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event %s: %w", event.EventType(), err)
	}
	return data, nil
}
