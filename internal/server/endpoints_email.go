package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/email"
	"github.com/godatei/datei/pkg/api"
)

type mailServer struct {
	svc *email.Service
}

// ListMailAccounts implements [StrictServerInterface].
func (s *mailServer) ListMailAccounts(
	ctx context.Context, request ListMailAccountsRequestObject,
) (ListMailAccountsResponseObject, error) {
	limit, offset := pagingParams(request.Params.Limit, request.Params.Offset)
	out, err := s.svc.ListAccounts(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return ListMailAccounts200JSONResponse(api.ListMailAccountsResponse{
		Items: out.Items,
		Total: out.Total,
	}), nil
}

// CreateMailAccount implements [StrictServerInterface].
func (s *mailServer) CreateMailAccount(
	ctx context.Context, request CreateMailAccountRequestObject,
) (CreateMailAccountResponseObject, error) {
	if request.Body == nil {
		return CreateMailAccount400JSONResponse{Message: "request body is required"}, nil
	}
	result, err := s.svc.CreateAccount(ctx, email.CreateAccountInput{
		Name:     request.Body.Name,
		ImapHost: request.Body.ImapHost,
		ImapPort: request.Body.ImapPort,
		Username: request.Body.Username,
		Password: request.Body.Password,
		Security: email.Security(request.Body.Security),
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInvalidInput), errors.Is(err, apperrors.ErrConnectionTestFailed):
			return CreateMailAccount400JSONResponse{Message: err.Error()}, nil
		case errors.Is(err, apperrors.ErrConcurrentUpdate):
			return CreateMailAccount409Response{}, nil
		}
		return nil, err
	}
	return CreateMailAccount201JSONResponse(*result), nil
}

// GetMailAccount implements [StrictServerInterface].
func (s *mailServer) GetMailAccount(
	ctx context.Context, request GetMailAccountRequestObject,
) (GetMailAccountResponseObject, error) {
	result, err := s.svc.GetAccount(ctx, request.AccountId)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return GetMailAccount404Response{}, nil
		}
		return nil, err
	}
	return GetMailAccount200JSONResponse(*result), nil
}

// UpdateMailAccount implements [StrictServerInterface].
func (s *mailServer) UpdateMailAccount(
	ctx context.Context, request UpdateMailAccountRequestObject,
) (UpdateMailAccountResponseObject, error) {
	if request.Body == nil {
		return UpdateMailAccount400JSONResponse{Message: "request body is required"}, nil
	}
	result, err := s.svc.UpdateAccount(ctx, request.AccountId, email.UpdateAccountInput{
		Name:     request.Body.Name,
		ImapHost: request.Body.ImapHost,
		ImapPort: request.Body.ImapPort,
		Username: request.Body.Username,
		Password: request.Body.Password,
		Security: email.Security(request.Body.Security),
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			return UpdateMailAccount404Response{}, nil
		case errors.Is(err, apperrors.ErrInvalidInput), errors.Is(err, apperrors.ErrConnectionTestFailed):
			return UpdateMailAccount400JSONResponse{Message: err.Error()}, nil
		case errors.Is(err, apperrors.ErrConcurrentUpdate):
			return UpdateMailAccount409Response{}, nil
		}
		return nil, err
	}
	return UpdateMailAccount200JSONResponse(*result), nil
}

// DeleteMailAccount implements [StrictServerInterface].
func (s *mailServer) DeleteMailAccount(
	ctx context.Context, request DeleteMailAccountRequestObject,
) (DeleteMailAccountResponseObject, error) {
	err := s.svc.DeleteAccount(ctx, request.AccountId)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			return DeleteMailAccount404Response{}, nil
		case errors.Is(err, apperrors.ErrConcurrentUpdate):
			return DeleteMailAccount409Response{}, nil
		}
		return nil, err
	}
	return DeleteMailAccount204Response{}, nil
}

// TestMailAccount implements [StrictServerInterface].
func (s *mailServer) TestMailAccount(
	ctx context.Context, request TestMailAccountRequestObject,
) (TestMailAccountResponseObject, error) {
	result, err := s.svc.TestConnection(ctx, request.AccountId)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return TestMailAccount404Response{}, nil
		}
		return nil, err
	}
	return TestMailAccount200JSONResponse(*result), nil
}

// ListAllMailRules implements [StrictServerInterface].
func (s *mailServer) ListAllMailRules(
	ctx context.Context, request ListAllMailRulesRequestObject,
) (ListAllMailRulesResponseObject, error) {
	limit, offset := pagingParams(request.Params.Limit, request.Params.Offset)
	out, err := s.svc.ListAllRules(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return ListAllMailRules200JSONResponse(api.ListMailRulesResponse{
		Items: out.Items,
		Total: out.Total,
	}), nil
}

// CreateMailRule implements [StrictServerInterface].
//
//nolint:dupl // create/update return distinct generated response types; cannot share the error mapping
func (s *mailServer) CreateMailRule(
	ctx context.Context, request CreateMailRuleRequestObject,
) (CreateMailRuleResponseObject, error) {
	if request.Body == nil {
		return CreateMailRule400Response{}, nil
	}
	result, err := s.svc.CreateRule(ctx, request.Body.AccountId, email.RuleInput{
		Name:              request.Body.Name,
		Order:             request.Body.Order,
		Enabled:           request.Body.Enabled,
		Folder:            request.Body.Folder,
		FilterFrom:        request.Body.FilterFrom,
		FilterSubject:     request.Body.FilterSubject,
		MaxAgeDays:        request.Body.MaxAgeDays,
		AttachmentPattern: request.Body.AttachmentPattern,
		Action:            email.Action(request.Body.Action),
		TargetDirectoryID: request.Body.TargetDirectoryId,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			return CreateMailRule404Response{}, nil
		case errors.Is(err, apperrors.ErrInvalidInput):
			return CreateMailRule400Response{}, nil
		case errors.Is(err, apperrors.ErrConcurrentUpdate):
			return CreateMailRule409Response{}, nil
		}
		return nil, err
	}
	return CreateMailRule201JSONResponse(*result), nil
}

// GetMailRule implements [StrictServerInterface].
func (s *mailServer) GetMailRule(
	ctx context.Context, request GetMailRuleRequestObject,
) (GetMailRuleResponseObject, error) {
	result, err := s.svc.GetRule(ctx, request.RuleId)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return GetMailRule404Response{}, nil
		}
		return nil, err
	}
	return GetMailRule200JSONResponse(*result), nil
}

// UpdateMailRule implements [StrictServerInterface].
//
//nolint:dupl // create/update return distinct generated response types; cannot share the error mapping
func (s *mailServer) UpdateMailRule(
	ctx context.Context, request UpdateMailRuleRequestObject,
) (UpdateMailRuleResponseObject, error) {
	if request.Body == nil {
		return UpdateMailRule400Response{}, nil
	}
	result, err := s.svc.UpdateRule(ctx, request.RuleId, request.Body.AccountId, email.RuleInput{
		Name:              request.Body.Name,
		Order:             request.Body.Order,
		Enabled:           request.Body.Enabled,
		Folder:            request.Body.Folder,
		FilterFrom:        request.Body.FilterFrom,
		FilterSubject:     request.Body.FilterSubject,
		MaxAgeDays:        request.Body.MaxAgeDays,
		AttachmentPattern: request.Body.AttachmentPattern,
		Action:            email.Action(request.Body.Action),
		TargetDirectoryID: request.Body.TargetDirectoryId,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			return UpdateMailRule404Response{}, nil
		case errors.Is(err, apperrors.ErrInvalidInput):
			return UpdateMailRule400Response{}, nil
		case errors.Is(err, apperrors.ErrConcurrentUpdate):
			return UpdateMailRule409Response{}, nil
		}
		return nil, err
	}
	return UpdateMailRule200JSONResponse(*result), nil
}

// DeleteMailRule implements [StrictServerInterface].
func (s *mailServer) DeleteMailRule(
	ctx context.Context, request DeleteMailRuleRequestObject,
) (DeleteMailRuleResponseObject, error) {
	err := s.svc.DeleteRule(ctx, request.RuleId)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			return DeleteMailRule404Response{}, nil
		case errors.Is(err, apperrors.ErrConcurrentUpdate):
			return DeleteMailRule409Response{}, nil
		}
		return nil, err
	}
	return DeleteMailRule204Response{}, nil
}

func pagingParams(limit, offset *int) (int, int) {
	l := 100
	if limit != nil {
		l = *limit
	}
	o := 0
	if offset != nil {
		o = *offset
	}
	return l, o
}
