import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { BehaviorSubject, of } from 'rxjs';

import { Datei } from '~/api/models';
import { DashboardComponent } from '~/frontend/dashboard/dashboard.component';

const makeDir = (id: string): Datei => ({
  id,
  name: 'folder',
  isDirectory: true,
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
});

const makeFile = (id: string): Datei => ({
  id,
  name: 'file.txt',
  isDirectory: false,
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
});

const EMPTY_LIST = { items: [], total: 0 };

describe('DashboardComponent', () => {
  let component: DashboardComponent;
  let fixture: ComponentFixture<DashboardComponent>;
  let httpTesting: HttpTestingController;
  let routerNavigate: ReturnType<typeof vi.fn>;
  let dialogOpen: ReturnType<typeof vi.fn>;
  let queryParamMap$: BehaviorSubject<ReturnType<typeof convertToParamMap>>;

  beforeEach(async () => {
    queryParamMap$ = new BehaviorSubject(convertToParamMap({}));
    routerNavigate = vi.fn();
    dialogOpen = vi.fn();

    await TestBed.configureTestingModule({
      imports: [DashboardComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { queryParamMap: queryParamMap$ } },
        { provide: Router, useValue: { navigate: routerNavigate } },
        { provide: MatDialog, useValue: { open: dialogOpen } },
        { provide: MatSnackBar, useValue: { open: vi.fn() } },
      ],
    }).compileComponents();

    httpTesting = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(DashboardComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should compile', () => {
    httpTesting.expectOne('/api/v1/datei').flush(EMPTY_LIST);
    expect(component).toBeTruthy();
  });

  describe('row double-click', () => {
    it('navigates into a directory', () => {
      httpTesting.expectOne('/api/v1/datei').flush(EMPTY_LIST);
      component['onRowDblClick'](makeDir('dir-123'));
      expect(routerNavigate).toHaveBeenCalledWith(
        [],
        expect.objectContaining({ queryParams: { parentId: 'dir-123' } }),
      );
    });

    it('does not navigate for a file', () => {
      httpTesting.expectOne('/api/v1/datei').flush(EMPTY_LIST);
      component['onRowDblClick'](makeFile('file-123'));
      expect(routerNavigate).not.toHaveBeenCalled();
    });
  });

  describe('listDatei params', () => {
    it('omits parentId query param at the root', () => {
      const req = httpTesting.expectOne((r) => r.url === '/api/v1/datei' && !r.params.has('parentId'));
      expect(req.request.method).toBe('GET');
      req.flush(EMPTY_LIST);
    });

    it('includes parentId as a query param when navigating into a subdirectory', () => {
      httpTesting.expectOne('/api/v1/datei').flush(EMPTY_LIST);

      queryParamMap$.next(convertToParamMap({ parentId: 'dir-456' }));
      fixture.detectChanges();

      const req = httpTesting.expectOne((r) => r.url === '/api/v1/datei' && r.params.get('parentId') === 'dir-456');
      expect(req.request.method).toBe('GET');
      req.flush(EMPTY_LIST);
      httpTesting.expectOne('/api/v1/datei/dir-456/path').flush([]);
    });
  });

  describe('createDatei body', () => {
    it('includes parentId in the multipart body when in a subdirectory', () => {
      httpTesting.expectOne('/api/v1/datei').flush(EMPTY_LIST);

      queryParamMap$.next(convertToParamMap({ parentId: 'dir-789' }));
      fixture.detectChanges();
      httpTesting.expectOne((r) => r.url === '/api/v1/datei' && r.params.get('parentId') === 'dir-789').flush(EMPTY_LIST);
      httpTesting.expectOne('/api/v1/datei/dir-789/path').flush([]);

      dialogOpen.mockReturnValue({ afterClosed: () => of('New Folder') });
      component['openNewFolderDialog']();

      const req = httpTesting.expectOne((r) => r.url === '/api/v1/datei' && r.method === 'POST');
      expect(req.request.body.get('name')).toBe('New Folder');
      expect(req.request.body.get('parentId')).toBe('dir-789');
      req.flush(makeDir('new-dir'));
    });

    it('omits parentId from the multipart body at the root', () => {
      httpTesting.expectOne('/api/v1/datei').flush(EMPTY_LIST);

      dialogOpen.mockReturnValue({ afterClosed: () => of('Root Folder') });
      component['openNewFolderDialog']();

      const req = httpTesting.expectOne((r) => r.url === '/api/v1/datei' && r.method === 'POST');
      expect(req.request.body.get('name')).toBe('Root Folder');
      expect(req.request.body.get('parentId')).toBeNull();
      req.flush(makeDir('new-dir'));
    });
  });
});
