import { App } from '@/app/app';
import { appConfig } from '@/app/app.config';
import { bootstrapApplication } from '@angular/platform-browser';

bootstrapApplication(App, appConfig).catch((error: unknown) => console.error(error));
