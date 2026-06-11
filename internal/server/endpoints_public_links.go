package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/link"
	"github.com/godatei/datei/pkg/api"
)

// UnlockPublicLink implements [StrictServerInterface]. It is the only
// unauthenticated endpoint in this file; success returns a short-lived JWT
// that the viewer presents on subsequent list/download calls.
func (s *server) UnlockPublicLink(
	ctx context.Context,
	request UnlockPublicLinkRequestObject,
) (UnlockPublicLinkResponseObject, error) {
	code := ""
	if request.Body != nil && request.Body.Code != nil {
		code = *request.Body.Code
	}

	result, err := s.publicLinkService.UnlockPublicLink(ctx, request.Key, code)
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrLinkCodeRequired):
			return UnlockPublicLink403Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkNotFound),
			errors.Is(err, dateierrors.ErrLinkRevoked),
			errors.Is(err, dateierrors.ErrLinkExpired):
			return UnlockPublicLink404Response{}, nil
		default:
			return nil, err
		}
	}

	return UnlockPublicLink200JSONResponse(api.UnlockPublicLinkResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
	}), nil
}

// ListPublicLinkDateien implements [StrictServerInterface]. The session claims
// are extracted from the public-link JWT by the auth middleware and read here
// from ctx.
func (s *server) ListPublicLinkDateien(
	ctx context.Context,
	request ListPublicLinkDateienRequestObject,
) (ListPublicLinkDateienResponseObject, error) {
	session := link.RequirePublicLinkSessionFromContext(ctx)

	result, err := s.publicLinkService.ListPublicLinkDateien(ctx, session, request.Params.ParentId)
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrLinkUnauthorized):
			return ListPublicLinkDateien401Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkExpired),
			errors.Is(err, dateierrors.ErrLinkRevoked),
			errors.Is(err, dateierrors.ErrLinkDateiNotShared):
			return ListPublicLinkDateien403Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkNotFound),
			errors.Is(err, dateierrors.ErrNotFound):
			return ListPublicLinkDateien404Response{}, nil
		default:
			return nil, err
		}
	}

	return ListPublicLinkDateien200JSONResponse(api.ListPublicLinkDateienResponse{
		Name:      result.Name,
		OwnerName: result.OwnerName,
		ExpiresAt: result.ExpiresAt,
		Items:     result.Items,
	}), nil
}

// DownloadPublicLinkDatei implements [StrictServerInterface].
func (s *server) DownloadPublicLinkDatei(
	ctx context.Context,
	request DownloadPublicLinkDateiRequestObject,
) (DownloadPublicLinkDateiResponseObject, error) {
	session := link.RequirePublicLinkSessionFromContext(ctx)

	result, err := s.publicLinkService.DownloadPublicLinkDatei(ctx, session, request.DateiId)
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrIsDirectory):
			return DownloadPublicLinkDatei409Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkUnauthorized):
			return DownloadPublicLinkDatei401Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkExpired),
			errors.Is(err, dateierrors.ErrLinkRevoked),
			errors.Is(err, dateierrors.ErrLinkDateiNotShared):
			return DownloadPublicLinkDatei403Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkNotFound),
			errors.Is(err, dateierrors.ErrNotFound),
			errors.Is(err, dateierrors.ErrNoContent):
			return DownloadPublicLinkDatei404Response{}, nil
		default:
			return nil, err
		}
	}

	return DownloadPublicLinkDatei200ApplicationoctetStreamResponse{
		Body: result.Reader,
		Headers: DownloadPublicLinkDatei200ResponseHeaders{
			ContentDisposition: attachmentDisposition(result.ContentFileName),
			ContentType:        result.ContentType,
		},
		ContentLength: result.ContentLength,
	}, nil
}
