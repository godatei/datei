package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/pkg/api"
)

// ListPublicLinkDateien implements [StrictServerInterface].
func (s *server) ListPublicLinkDateien(
	ctx context.Context,
	request ListPublicLinkDateienRequestObject,
) (ListPublicLinkDateienResponseObject, error) {
	code := ""
	if request.Params.XDateiLinkCode != nil {
		code = *request.Params.XDateiLinkCode
	}

	result, err := s.linkService.ListPublicLinkDateien(ctx, request.AccessToken, request.Params.ParentId, code)
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrLinkCodeRequired):
			return ListPublicLinkDateien403Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkExpired):
			return ListPublicLinkDateien410Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkNotFound),
			errors.Is(err, dateierrors.ErrLinkRevoked),
			errors.Is(err, dateierrors.ErrLinkDateiNotShared):
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
	code := ""
	if request.Params.XDateiLinkCode != nil {
		code = *request.Params.XDateiLinkCode
	}

	result, err := s.linkService.DownloadPublicLinkDatei(ctx, request.AccessToken, request.DateiId, code)
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrLinkCodeRequired):
			return DownloadPublicLinkDatei403Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkExpired):
			return DownloadPublicLinkDatei410Response{}, nil
		case errors.Is(err, dateierrors.ErrIsDirectory):
			return DownloadPublicLinkDatei409Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkNotFound),
			errors.Is(err, dateierrors.ErrLinkRevoked),
			errors.Is(err, dateierrors.ErrLinkDateiNotShared),
			errors.Is(err, dateierrors.ErrNotFound):
			return DownloadPublicLinkDatei404Response{}, nil
		default:
			return nil, err
		}
	}

	return DownloadPublicLinkDatei200ApplicationoctetStreamResponse{
		Body: result.Reader,
		Headers: DownloadPublicLinkDatei200ResponseHeaders{
			ContentDisposition: fmt.Sprintf(`attachment; filename="%v"`, result.ContentFileName),
			ContentType:        result.ContentType,
		},
		ContentLength: result.ContentLength,
	}, nil
}
