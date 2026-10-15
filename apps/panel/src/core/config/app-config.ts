import { HttpBackend, HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { firstValueFrom } from 'rxjs';

// Settings that differ between environments. The panel is static files, so
// they come from /config.json, which the container writes from its
// environment at start (docker-entrypoint.sh): one build for every
// environment. public/config.json holds the local defaults.
export interface AppConfig {
  // The public API (services/gateway).
  apiBaseUrl: string;
}

@Injectable({ providedIn: 'root' })
export class ConfigStore {
  // HttpBackend skips the interceptors, which need the config themselves.
  private readonly http = new HttpClient(inject(HttpBackend));
  private value: AppConfig | null = null;

  get config(): AppConfig {
    if (!this.value) throw new Error('config.json is not loaded yet');
    return this.value;
  }

  async load(): Promise<void> {
    this.value = await firstValueFrom(this.http.get<AppConfig>('/config.json'));
  }
}
