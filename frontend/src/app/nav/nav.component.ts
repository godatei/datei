import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NgOptimizedImage } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatRippleModule } from '@angular/material/core';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatSidenav, MatSidenavModule } from '@angular/material/sidenav';
import { MatToolbarModule } from '@angular/material/toolbar';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';
import { UserAvatarComponent } from '~/frontend/users/user-avatar.component';

interface NavItem {
  label: string;
  icon: string;
  route: string;
  exact: boolean;
}

@Component({
  selector: 'app-nav',
  templateUrl: './nav.component.html',
  styleUrl: './nav.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    NgOptimizedImage,
    MatToolbarModule,
    MatButtonModule,
    MatSidenavModule,
    MatIconModule,
    MatMenuModule,
    MatRippleModule,
    RouterOutlet,
    RouterLink,
    RouterLinkActive,
    UserAvatarComponent,
  ],
})
export class NavComponent {
  private readonly breakpointObserver = inject(BreakpointObserver);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  private readonly drawer = viewChild.required(MatSidenav);

  private readonly handsetObserver = toSignal(this.breakpointObserver.observe(Breakpoints.Handset));
  protected readonly isHandset = computed(() => this.handsetObserver()?.matches ?? false);
  protected readonly userName = computed(() => this.auth.userName() ?? 'User');
  protected readonly isAdmin = this.auth.isAdmin;

  protected readonly navItems: readonly NavItem[] = [
    { label: 'My files', icon: 'folder', route: '/', exact: true },
    { label: 'Links', icon: 'link', route: '/links', exact: false },
    { label: 'Trash', icon: 'delete', route: '/trash', exact: false },
  ];

  protected readonly expanded = signal(false);
  // On handset the rail is shown as a modal expanded rail (replaces the drawer).
  protected readonly railExpanded = computed(() => this.expanded() || this.isHandset());

  protected toggleExpanded() {
    this.expanded.update((value) => !value);
  }

  protected onNavItemClick() {
    if (this.isHandset()) {
      this.drawer().close();
    }
  }

  logout() {
    this.auth.logout();
    this.router.navigate(['/login']);
  }
}
