package events

import (
	"encoding/json"
	"fmt"
)

// Deserialize unmarshals event data from JSON using the event type
func Deserialize(eventType string, data []byte) (DomainEvent, error) {
	switch eventType {
	case "DateiCreated":
		var event DateiCreatedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiCreatedEvent: %w", err)
		}
		return event, nil

	case "DateiRenamed":
		var event DateiRenamedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiRenamedEvent: %w", err)
		}
		return event, nil

	case "DateiVersionUploaded":
		var event DateiVersionUploadedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiVersionUploadedEvent: %w", err)
		}
		return event, nil

	case "DateiMoved":
		var event DateiMovedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiMovedEvent: %w", err)
		}
		return event, nil

	case "DateiTrashed":
		var event DateiTrashedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiTrashedEvent: %w", err)
		}
		return event, nil

	case "DateiRestored":
		var event DateiRestoredEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiRestoredEvent: %w", err)
		}
		return event, nil

	case "DateiLinked":
		var event DateiLinkedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiLinkedEvent: %w", err)
		}
		return event, nil

	case "DateiUnlinked":
		var event DateiUnlinkedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiUnlinkedEvent: %w", err)
		}
		return event, nil

	case "DateiPermissionGranted":
		var event DateiPermissionGrantedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiPermissionGrantedEvent: %w", err)
		}
		return event, nil

	case "DateiPermissionRevoked":
		var event DateiPermissionRevokedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DateiPermissionRevokedEvent: %w", err)
		}
		return event, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
}

// Serialize marshals an event to JSON
func Serialize(event DomainEvent) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event %s: %w", event.EventType(), err)
	}
	return data, nil
}
