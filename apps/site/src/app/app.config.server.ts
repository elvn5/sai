import { appConfig } from '@/app/app.config';
import { serverRoutes } from '@/app/app.routes.server';
import { APP_CONFIG, SERVER_API_URL } from '@/core/config/app-config';
import { HTTP_TRANSFER_CACHE_ORIGIN_MAP } from '@angular/common/http';
import { type ApplicationConfig, inject, mergeApplicationConfig } from '@angular/core';
import { provideServerRendering, withRoutes } from '@angular/ssr';

const serverConfig: ApplicationConfig = {
  providers: [
    provideServerRendering(withRoutes(serverRoutes)),
    // The server calls the API over the private network; the browser finds
    // those responses in the transfer cache under the public URL.
    {
      provide: HTTP_TRANSFER_CACHE_ORIGIN_MAP,
      useFactory: (): Record<string, string> => {
        const internal = new URL(inject(SERVER_API_URL)).origin;
        const external = new URL(inject(APP_CONFIG).apiBaseUrl).origin;
        return internal === external ? {} : { [internal]: external };
      },
    },
  ],
};

export const config = mergeApplicationConfig(appConfig, serverConfig);
