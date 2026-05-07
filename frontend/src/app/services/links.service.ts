import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable, map } from 'rxjs';
import {
  addDateiToLink,
  createLink,
  downloadPublicLinkDatei,
  getLink,
  listLinks,
  listPublicLinkDateien,
  removeDateiFromLink,
  revokeLink,
  rotateLinkAccessToken,
  updateLink,
} from '~/api/functions';
import type { CreateLinkRequest } from '~/api/models/create-link-request';
import type { Link } from '~/api/models/link';
import type { LinkDetail } from '~/api/models/link-detail';
import type { ListPublicLinkDateienResponse } from '~/api/models/list-public-link-dateien-response';
import type { UpdateLinkRequest } from '~/api/models/update-link-request';

@Injectable({ providedIn: 'root' })
export class LinksService {
  private readonly httpClient = inject(HttpClient);

  // ============================================================================
  // Owner-side
  // ============================================================================

  listLinks(): Observable<Link[]> {
    return listLinks(this.httpClient, '').pipe(map((r) => r.body.items));
  }

  getLink(id: string): Observable<LinkDetail> {
    return getLink(this.httpClient, '', { id }).pipe(map((r) => r.body));
  }

  createLink(body: CreateLinkRequest): Observable<LinkDetail> {
    return createLink(this.httpClient, '', { body }).pipe(map((r) => r.body));
  }

  updateLink(id: string, body: UpdateLinkRequest): Observable<LinkDetail> {
    return updateLink(this.httpClient, '', { id, body }).pipe(map((r) => r.body));
  }

  revokeLink(id: string): Observable<void> {
    return revokeLink(this.httpClient, '', { id }).pipe(map(() => undefined));
  }

  rotateAccessToken(id: string): Observable<LinkDetail> {
    return rotateLinkAccessToken(this.httpClient, '', { id }).pipe(map((r) => r.body));
  }

  addDatei(linkId: string, dateiId: string): Observable<LinkDetail> {
    return addDateiToLink(this.httpClient, '', { id: linkId, body: { dateiId } }).pipe(
      map((r) => r.body),
    );
  }

  removeDatei(linkId: string, dateiId: string): Observable<void> {
    return removeDateiFromLink(this.httpClient, '', { id: linkId, dateiId }).pipe(
      map(() => undefined),
    );
  }

  // ============================================================================
  // Public-side
  // ============================================================================

  listPublicDateien(
    accessToken: string,
    parentId: string | undefined,
    code: string | undefined,
  ): Observable<ListPublicLinkDateienResponse> {
    return listPublicLinkDateien(this.httpClient, '', {
      accessToken,
      parentId,
      'X-Datei-Link-Code': code,
    }).pipe(map((r) => r.body));
  }

  downloadPublicDatei(
    accessToken: string,
    dateiId: string,
    code: string | undefined,
  ): Observable<Blob> {
    return downloadPublicLinkDatei(this.httpClient, '', {
      accessToken,
      dateiId,
      'X-Datei-Link-Code': code,
    }).pipe(
      // The generated function types the body as `any` (it streams
      // application/octet-stream) but the runtime value is a Blob.
      map((r) => r.body as unknown as Blob),
    );
  }

  buildShareUrl(accessToken: string): string {
    return `${window.location.origin}/share/${accessToken}`;
  }
}
