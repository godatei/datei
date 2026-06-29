import { Component, input } from '@angular/core';
import { NgOptimizedImage } from '@angular/common';
import { MatCardModule } from '@angular/material/card';

/** Shared wrapper for the unauthenticated pages (login, register, reset, …):
 *  centers a Material card on the auth surface and renders the branded header
 *  (datei logo, title, description). Pages project their form/content via
 *  `<ng-content>`. */
@Component({
  selector: 'app-auth-layout',
  imports: [MatCardModule, NgOptimizedImage],
  templateUrl: './auth-layout.component.html',
  styleUrl: './auth-layout.component.scss',
})
export class AuthLayoutComponent {
  readonly title = input.required<string>();
  readonly description = input<string>();
}
