import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { firstValueFrom } from 'rxjs';

// Image uploads through the API (services/media). Returns the file's URL.
@Injectable({ providedIn: 'root' })
export class UploadApi {
  private readonly http = inject(HttpClient);

  async uploadImage(file: File): Promise<string> {
    const body = new FormData();
    body.append('file', file);
    const { url } = await firstValueFrom(this.http.post<{ url: string }>('/uploads/image', body));
    return url;
  }
}
