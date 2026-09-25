// Direct-to-R2 uploads: this service only ever hands out presigned URLs —
// the actual file bytes go straight from the browser to R2 and never touch
// this server's memory or disk. That means the R2 bucket must allow
// cross-origin PUT requests from the frontend origin(s), which S3-compatible
// buckets don't by default.
//
// Configure this once per bucket in the Cloudflare dashboard under
// R2 > <bucket> > Settings > CORS Policy (or via `wrangler r2 bucket cors put`):
//
//   [
//     {
//       "AllowedOrigins": ["https://ermekten.com", "https://test.ermekten.com"],
//       "AllowedMethods": ["PUT", "GET"],
//       "AllowedHeaders": ["Content-Type"],
//       "MaxAgeSeconds": 3600
//     }
//   ]
//
// Without this, the browser's PUT to the presigned URL is blocked by CORS
// before it ever reaches R2, even though the URL itself is valid.
import { randomUUID } from 'node:crypto';
import { extname } from 'node:path';
import {
  DeleteObjectCommand,
  GetObjectCommand,
  PutObjectCommand,
  S3Client,
} from '@aws-sdk/client-s3';
import { getSignedUrl } from '@aws-sdk/s3-request-presigner';
import { Injectable, InternalServerErrorException, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

const UPLOAD_URL_EXPIRY_SECONDS = 600;
const DOWNLOAD_URL_EXPIRY_SECONDS = 3600;

export interface PresignedUpload {
  uploadUrl: string;
  key: string;
  publicUrl: string;
}

@Injectable()
export class StorageService {
  private readonly logger = new Logger(StorageService.name);
  private client: S3Client | null = null;

  constructor(private readonly configService: ConfigService) {}

  async generatePresignedUploadUrl(
    filename: string,
    contentType: string,
    folder?: string,
  ): Promise<PresignedUpload> {
    const bucket = this.getBucketName();
    const key = this.buildObjectKey(filename, folder);

    const command = new PutObjectCommand({ Bucket: bucket, Key: key, ContentType: contentType });

    let uploadUrl: string;
    try {
      uploadUrl = await getSignedUrl(this.getClient(), command, {
        expiresIn: UPLOAD_URL_EXPIRY_SECONDS,
      });
    } catch (error) {
      this.logger.error(`Failed to create presigned upload URL for key "${key}"`, error);
      throw new InternalServerErrorException('Could not create upload URL');
    }

    return { uploadUrl, key, publicUrl: this.buildPublicUrl(key) };
  }

  async deleteFile(key: string): Promise<void> {
    const bucket = this.getBucketName();
    try {
      await this.getClient().send(new DeleteObjectCommand({ Bucket: bucket, Key: key }));
    } catch (error) {
      this.logger.error(`Failed to delete R2 object "${key}"`, error);
      throw new InternalServerErrorException('Could not delete file');
    }
  }

  async getPresignedDownloadUrl(
    key: string,
    expiresIn: number = DOWNLOAD_URL_EXPIRY_SECONDS,
  ): Promise<string> {
    const bucket = this.getBucketName();
    const command = new GetObjectCommand({ Bucket: bucket, Key: key });

    try {
      return await getSignedUrl(this.getClient(), command, { expiresIn });
    } catch (error) {
      this.logger.error(`Failed to create presigned download URL for key "${key}"`, error);
      throw new InternalServerErrorException('Could not create download URL');
    }
  }

  private buildObjectKey(filename: string, folder?: string): string {
    const extension = extname(filename).toLowerCase();
    const key = `${randomUUID()}${extension}`;
    return folder ? `${folder}/${key}` : key;
  }

  private buildPublicUrl(key: string): string {
    const publicUrl = this.configService.get<string>('R2_PUBLIC_URL');
    if (!publicUrl) {
      this.logger.error('R2_PUBLIC_URL is not configured — cannot build a public file URL');
      throw new InternalServerErrorException('Storage service is not configured');
    }
    return `${publicUrl.replace(/\/$/, '')}/${key}`;
  }

  private getBucketName(): string {
    const bucket = this.configService.get<string>('R2_BUCKET_NAME');
    if (!bucket) {
      this.logger.error('R2_BUCKET_NAME is not configured');
      throw new InternalServerErrorException('Storage service is not configured');
    }
    return bucket;
  }

  private getClient(): S3Client {
    if (this.client) return this.client;

    const accountId = this.configService.get<string>('R2_ACCOUNT_ID');
    const accessKeyId = this.configService.get<string>('R2_ACCESS_KEY_ID');
    const secretAccessKey = this.configService.get<string>('R2_SECRET_ACCESS_KEY');

    if (!accountId || !accessKeyId || !secretAccessKey) {
      this.logger.error(
        'R2 credentials are not fully configured (R2_ACCOUNT_ID / R2_ACCESS_KEY_ID / R2_SECRET_ACCESS_KEY)',
      );
      throw new InternalServerErrorException('Storage service is not configured');
    }

    this.client = new S3Client({
      region: 'auto',
      endpoint: `https://${accountId}.r2.cloudflarestorage.com`,
      credentials: { accessKeyId, secretAccessKey },
    });
    return this.client;
  }
}
