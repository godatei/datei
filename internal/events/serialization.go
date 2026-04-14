package events

import (
	"encoding/json"
	"fmt"
	"reflect"
)

var eventFactories = map[string]func() DomainEvent{
	"DateiCreated":                    func() DomainEvent { return &DateiCreatedEvent{} },
	"DateiRenamed":                    func() DomainEvent { return &DateiRenamedEvent{} },
	"DateiVersionUploaded":            func() DomainEvent { return &DateiVersionUploadedEvent{} },
	"DateiMoved":                      func() DomainEvent { return &DateiMovedEvent{} },
	"DateiTrashed":                    func() DomainEvent { return &DateiTrashedEvent{} },
	"DateiRestored":                   func() DomainEvent { return &DateiRestoredEvent{} },
	"DateiLinked":                     func() DomainEvent { return &DateiLinkedEvent{} },
	"DateiUnlinked":                   func() DomainEvent { return &DateiUnlinkedEvent{} },
	"DateiPermissionGranted":          func() DomainEvent { return &DateiPermissionGrantedEvent{} },
	"DateiPermissionRevoked":          func() DomainEvent { return &DateiPermissionRevokedEvent{} },
	"UserRegistered":                  func() DomainEvent { return &UserRegisteredEvent{} },
	"UserNameChanged":                 func() DomainEvent { return &UserNameChangedEvent{} },
	"UserPasswordChanged":             func() DomainEvent { return &UserPasswordChangedEvent{} },
	"UserEmailChanged":                func() DomainEvent { return &UserEmailChangedEvent{} },
	"UserEmailVerified":               func() DomainEvent { return &UserEmailVerifiedEvent{} },
	"UserEmailAdded":                  func() DomainEvent { return &UserEmailAddedEvent{} },
	"UserEmailRemoved":                func() DomainEvent { return &UserEmailRemovedEvent{} },
	"UserEmailSetPrimary":             func() DomainEvent { return &UserEmailSetPrimaryEvent{} },
	"UserMFASetupInitiated":           func() DomainEvent { return &UserMFASetupInitiatedEvent{} },
	"UserMFAEnabled":                  func() DomainEvent { return &UserMFAEnabledEvent{} },
	"UserMFADisabled":                 func() DomainEvent { return &UserMFADisabledEvent{} },
	"UserMFARecoveryCodeUsed":         func() DomainEvent { return &UserMFARecoveryCodeUsedEvent{} },
	"UserMFARecoveryCodesRegenerated": func() DomainEvent { return &UserMFARecoveryCodesRegeneratedEvent{} },
	"UserArchived":                    func() DomainEvent { return &UserArchivedEvent{} },
	"UserLoggedIn":                    func() DomainEvent { return &UserLoggedInEvent{} },
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

	// Dereference the pointer to return a value type, preserving
	// compatibility with existing type switches (e.g. case DateiCreatedEvent:).
	return reflect.ValueOf(ptr).Elem().Interface().(DomainEvent), nil
}

// Serialize marshals an event to JSON
func Serialize(event DomainEvent) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event %s: %w", event.EventType(), err)
	}
	return data, nil
}
