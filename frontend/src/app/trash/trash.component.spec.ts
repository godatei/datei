import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router';
import { BehaviorSubject } from 'rxjs';

import { TrashComponent } from '~/frontend/trash/trash.component';

const EMPTY_TRASH = { items: [], total: 0 };

describe('TrashComponent', () => {
  let component: TrashComponent;
  let fixture: ComponentFixture<TrashComponent>;
  let httpTesting: HttpTestingController;
  let queryParamMap$: BehaviorSubject<ReturnType<typeof convertToParamMap>>;

  beforeEach(async () => {
    queryParamMap$ = new BehaviorSubject(convertToParamMap({}));

    await TestBed.configureTestingModule({
      imports: [TrashComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { queryParamMap: queryParamMap$ } },
        { provide: Router, useValue: { navigate: vi.fn() } },
      ],
    }).compileComponents();

    httpTesting = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(TrashComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should compile', () => {
    httpTesting.expectOne('/api/v1/trash').flush(EMPTY_TRASH);
    expect(component).toBeTruthy();
  });

  describe('listTrash params', () => {
    it('calls /trash at the root', () => {
      const req = httpTesting.expectOne('/api/v1/trash');
      expect(req.request.method).toBe('GET');
      req.flush(EMPTY_TRASH);
    });

    it('calls /trash/{id}/children when parentId is set', () => {
      httpTesting.expectOne('/api/v1/trash').flush(EMPTY_TRASH);

      queryParamMap$.next(convertToParamMap({ parentId: 'dir-123' }));
      fixture.detectChanges();

      const req = httpTesting.expectOne('/api/v1/trash/dir-123/children');
      expect(req.request.method).toBe('GET');
      req.flush(EMPTY_TRASH);
      httpTesting.expectOne('/api/v1/files/dir-123/path').flush([]);
    });
  });

  describe('path request', () => {
    it('does not request path at the root', () => {
      httpTesting.expectOne('/api/v1/trash').flush(EMPTY_TRASH);
      httpTesting.expectNone('/api/v1/files');
    });

    it('requests path for the current parentId', () => {
      httpTesting.expectOne('/api/v1/trash').flush(EMPTY_TRASH);

      queryParamMap$.next(convertToParamMap({ parentId: 'dir-456' }));
      fixture.detectChanges();

      httpTesting.expectOne('/api/v1/trash/dir-456/children').flush(EMPTY_TRASH);
      httpTesting.expectOne('/api/v1/files/dir-456/path').flush([]);
    });
  });
});
