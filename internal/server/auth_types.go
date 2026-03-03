package server

type LoginRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	MFACode  *string `json:"mfaCode,omitempty"`
}

type LoginResponse struct {
	Token       string `json:"token,omitempty"`
	RequiresMFA bool   `json:"requiresMfa,omitempty"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type ResetPasswordRequest struct {
	Email string `json:"email"`
}

type LoginConfigResponse struct {
	RegistrationEnabled bool `json:"registrationEnabled"`
}

type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty"`
	Password *string `json:"password,omitempty"`
}

type UpdateUserEmailRequest struct {
	Email string `json:"email"`
}

type SetupMFAResponse struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qrCodeUrl"`
}

type EnableMFARequest struct {
	Code string `json:"code"`
}

type EnableMFAResponse struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

type DisableMFARequest struct {
	Password string `json:"password"`
}

type RegenerateMFARecoveryCodesRequest struct {
	Password string `json:"password"`
}

type RegenerateMFARecoveryCodesResponse struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

type MFARecoveryCodesStatusResponse struct {
	RemainingCodes int32 `json:"remainingCodes"`
}
