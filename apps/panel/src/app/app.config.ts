import { routes } from '@/app/app.routes';
import { apiInterceptor } from '@/shared/api/api.interceptor';
import { ConfigStore } from '@/shared/config/app-config';
import { provideHttpClient, withFetch, withInterceptors } from '@angular/common/http';
import {
  type ApplicationConfig,
  inject,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
} from '@angular/core';
import { provideRouter } from '@angular/router';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideAppInitializer(() => inject(ConfigStore).load()),
    provideHttpClient(withFetch(), withInterceptors([apiInterceptor])),
    provideRouter(routes),
  ],
};
