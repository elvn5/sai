import type { GuideAttachment } from '@/core/api/types';
import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { firstValueFrom } from 'rxjs';

export const IMAGE_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp']);

interface PresignedUpload {
  uploadUrl: string;
  key: string;
  publicUrl: string;
}

// File uploads (services/media). Browser only.
@Injectable({ providedIn: 'root' })
export class UploadApi {
  private readonly http = inject(HttpClient);

  // Through the API; returns the image's URL (an /uploads path or R2).
  async uploadImage(file: File): Promise<string> {
    return (await firstValueFrom(this.http.post<{ url: string }>('/uploads/image', formData(file))))
      .url;
  }

  // A guide attachment (PDF, GPX…), with its name and size.
  uploadFile(file: File): Promise<GuideAttachment> {
    return firstValueFrom(this.http.post<GuideAttachment>('/uploads/file', formData(file)));
  }

  // Straight from the browser to the R2 bucket through a presigned URL
  // (gallery photos, which can be large). Returns the public URL.
  async uploadToStorage(file: File, folder: string): Promise<string> {
    if (!IMAGE_TYPES.has(file.type)) throw new Error('unsupported-file-type');
    const { uploadUrl, publicUrl } = await firstValueFrom(
      this.http.post<PresignedUpload>('/storage/upload-url', {
        filename: file.name,
        contentType: file.type,
        folder,
      }),
    );
    const response = await fetch(uploadUrl, {
      method: 'PUT',
      headers: { 'Content-Type': file.type },
      body: file,
    });
    if (!response.ok) throw new Error('upload-failed');
    return publicUrl;
  }
}

function formData(file: File): FormData {
  const body = new FormData();
  body.append('file', file);
  return body;
}
