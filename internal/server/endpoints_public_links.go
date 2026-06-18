package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/link"
	"github.com/godatei/datei/internal/linkauth"
	"github.com/godatei/datei/pkg/api"
)

type publicLinkServer struct {
	svc *link.PublicService
}

// UnlockPublicLink implements [StrictServerInterface]. It is the only
// unauthenticated endpoint in this file; success returns a short-lived JWT
// that the viewer presents on subsequent list/download calls.
func (s *publicLinkServer) UnlockPublicLink(
	ctx context.Context,
	request UnlockPublicLinkRequestObject,
) (UnlockPublicLinkResponseObject, error) {
	code := ""
	if request.Body != nil && request.Body.Code != nil {
		code = *request.Body.Code
	}

	result, err := s.svc.UnlockPublicLink(ctx, request.Key, code)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrLinkCodeRequired):
			return UnlockPublicLink403Response{}, nil
		case errors.Is(err, apperrors.ErrLinkNotFound),
			errors.Is(err, apperrors.ErrLinkRevoked),
			errors.Is(err, apperrors.ErrLinkExpired):
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

// ListPublicLinkFiles implements [StrictServerInterface]. The session claims
// are extracted from the public-link JWT by the auth middleware and read here
// from ctx.
func (s *publicLinkServer) ListPublicLinkFiles(
	ctx context.Context,
	request ListPublicLinkFilesRequestObject,
) (ListPublicLinkFilesResponseObject, error) {
	session := linkauth.RequirePublicLinkSessionFromContext(ctx)

	result, err := s.svc.ListPublicLinkFiles(ctx, session, request.Params.ParentId)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrLinkUnauthorized):
			return ListPublicLinkFiles401Response{}, nil
		case errors.Is(err, apperrors.ErrLinkExpired),
			errors.Is(err, apperrors.ErrLinkRevoked),
			errors.Is(err, apperrors.ErrLinkFileNotShared):
			return ListPublicLinkFiles403Response{}, nil
		case errors.Is(err, apperrors.ErrLinkNotFound),
			errors.Is(err, apperrors.ErrNotFound):
			return ListPublicLinkFiles404Response{}, nil
		default:
			return nil, err
		}
	}

	return ListPublicLinkFiles200JSONResponse(api.ListPublicLinkFilesResponse{
		Name:      result.Name,
		OwnerName: result.OwnerName,
		ExpiresAt: result.ExpiresAt,
		Items:     result.Items,
	}), nil
}

// DownloadPublicLinkFile implements [StrictServerInterface].
func (s *publicLinkServer) DownloadPublicLinkFile(
	ctx context.Context,
	request DownloadPublicLinkFileRequestObject,
) (DownloadPublicLinkFileResponseObject, error) {
	session := linkauth.RequirePublicLinkSessionFromContext(ctx)

	result, err := s.svc.DownloadPublicLinkFile(ctx, session, request.FileId)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrIsDirectory):
			return DownloadPublicLinkFile409Response{}, nil
		case errors.Is(err, apperrors.ErrLinkUnauthorized):
			return DownloadPublicLinkFile401Response{}, nil
		case errors.Is(err, apperrors.ErrLinkExpired),
			errors.Is(err, apperrors.ErrLinkRevoked),
			errors.Is(err, apperrors.ErrLinkFileNotShared):
			return DownloadPublicLinkFile403Response{}, nil
		case errors.Is(err, apperrors.ErrLinkNotFound),
			errors.Is(err, apperrors.ErrNotFound),
			errors.Is(err, apperrors.ErrNoContent):
			return DownloadPublicLinkFile404Response{}, nil
		default:
			return nil, err
		}
	}

	return DownloadPublicLinkFile200ApplicationoctetStreamResponse{
		Body: result.Reader,
		Headers: DownloadPublicLinkFile200ResponseHeaders{
			ContentDisposition: attachmentDisposition(result.ContentFileName),
			ContentType:        result.ContentType,
		},
		ContentLength: result.ContentLength,
	}, nil
}
