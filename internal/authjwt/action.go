package authjwt

import "fmt"

type Action string

const (
	ActionResetPassword Action = "reset-password"
	ActionVerifyEmail   Action = "verify-email"
)

func ParseAction(s string) (Action, error) {
	if s == "" {
		return "", nil
	}
	switch Action(s) {
	case ActionResetPassword, ActionVerifyEmail:
		return Action(s), nil
	default:
		return "", fmt.Errorf("unknown JWT action: %q", s)
	}
}
