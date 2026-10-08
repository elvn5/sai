import { useApiClient } from '~/shared/api/client';

export const ALLOWED_IMAGE_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp']);

interface PresignedUpload {
  uploadUrl: string;
  key: string;
  publicUrl: string;
}

// Direct-to-R2 upload: the file goes straight from the browser to the
// bucket via a presigned URL, never through the API server. Returns the
// file's public URL.
export function useImageUpload(): { uploadImage: (file: File, folder: string) => Promise<string> } {
  const api = useApiClient();

  async function uploadImage(file: File, folder: string): Promise<string> {
    if (!ALLOWED_IMAGE_TYPES.has(file.type)) {
      throw new Error('unsupported-file-type');
    }

    const { uploadUrl, publicUrl } = await api.post<PresignedUpload>('/storage/upload-url', {
      filename: file.name,
      contentType: file.type,
      folder,
    });

    const response = await fetch(uploadUrl, {
      method: 'PUT',
      headers: { 'Content-Type': file.type },
      body: file,
    });
    if (!response.ok) {
      throw new Error('upload-failed');
    }
    return publicUrl;
  }

  return { uploadImage };
}
