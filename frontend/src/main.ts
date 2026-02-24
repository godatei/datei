import { bootstrapApplication } from '@angular/platform-browser';
import { appConfig } from '~/frontend/app.config';
import { App } from '~/frontend/app';

bootstrapApplication(App, appConfig).catch((err) => console.error(err));
