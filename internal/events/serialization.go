package events

import (
	"encoding/json"
	"fmt"
)

// Deserialize unmarshals event data from JSON using the event type
//
//nolint:gocyclo // large switch is inherent to event type registry
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

	case "UserRegistered":
		var event UserRegisteredEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserRegisteredEvent: %w", err)
		}
		return event, nil

	case "UserNameChanged":
		var event UserNameChangedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserNameChangedEvent: %w", err)
		}
		return event, nil

	case "UserPasswordChanged":
		var event UserPasswordChangedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserPasswordChangedEvent: %w", err)
		}
		return event, nil

	case "UserEmailChanged":
		var event UserEmailChangedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserEmailChangedEvent: %w", err)
		}
		return event, nil

	case "UserEmailVerified":
		var event UserEmailVerifiedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserEmailVerifiedEvent: %w", err)
		}
		return event, nil

	case "UserEmailAdded":
		var event UserEmailAddedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserEmailAddedEvent: %w", err)
		}
		return event, nil

	case "UserEmailRemoved":
		var event UserEmailRemovedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserEmailRemovedEvent: %w", err)
		}
		return event, nil

	case "UserEmailSetPrimary":
		var event UserEmailSetPrimaryEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserEmailSetPrimaryEvent: %w", err)
		}
		return event, nil

	case "UserMFASetupInitiated":
		var event UserMFASetupInitiatedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserMFASetupInitiatedEvent: %w", err)
		}
		return event, nil

	case "UserMFAEnabled":
		var event UserMFAEnabledEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserMFAEnabledEvent: %w", err)
		}
		return event, nil

	case "UserMFADisabled":
		var event UserMFADisabledEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserMFADisabledEvent: %w", err)
		}
		return event, nil

	case "UserMFARecoveryCodeUsed":
		var event UserMFARecoveryCodeUsedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserMFARecoveryCodeUsedEvent: %w", err)
		}
		return event, nil

	case "UserMFARecoveryCodesRegenerated":
		var event UserMFARecoveryCodesRegeneratedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserMFARecoveryCodesRegeneratedEvent: %w", err)
		}
		return event, nil

	case "UserArchived":
		var event UserArchivedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserArchivedEvent: %w", err)
		}
		return event, nil

	case "UserLoggedIn":
		var event UserLoggedInEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal UserLoggedInEvent: %w", err)
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
