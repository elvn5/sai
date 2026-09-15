import { randomUUID } from 'node:crypto';
import { writeFile } from 'node:fs/promises';
import { extname, join } from 'node:path';
import { AnyAuthGuard } from '@/common/any-auth.guard';
import {
  BadRequestException,
  Controller,
  Post,
  UploadedFile,
  UseGuards,
  UseInterceptors,
} from '@nestjs/common';
import { FileInterceptor } from '@nestjs/platform-express';
import { diskStorage, memoryStorage } from 'multer';
import { convertImageBufferToWebp } from './image-conversion';
import { UPLOADS_DIR } from './uploads.constants';

const ALLOWED_IMAGE_MIME_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp', 'image/gif']);
const MAX_IMAGE_SIZE_BYTES = 8 * 1024 * 1024;

// General attachments for guides (maps, tracks, etc.) — not images, those
// go through /uploads/image instead.
const ALLOWED_FILE_MIME_TYPES = new Set([
  'application/pdf',
  'application/gpx+xml',
  'application/vnd.google-earth.kml+xml',
  'application/zip',
  'application/xml',
  'text/xml',
]);
const MAX_FILE_SIZE_BYTES = 20 * 1024 * 1024;

function randomFilename(originalname: string): string {
  return `${randomUUID()}${extname(originalname).toLowerCase()}`;
}

@Controller('uploads')
@UseGuards(AnyAuthGuard)
export class UploadsController {
  @Post('image')
  @UseInterceptors(
    FileInterceptor('file', {
      // Buffered in memory, not written to disk as-is — every upload gets
      // re-encoded to WebP below before it ever touches the filesystem.
      storage: memoryStorage(),
      limits: { fileSize: MAX_IMAGE_SIZE_BYTES },
      fileFilter: (_req, file, callback) => {
        if (!ALLOWED_IMAGE_MIME_TYPES.has(file.mimetype)) {
          callback(
            new BadRequestException('Only JPEG, PNG, WEBP or GIF images are allowed'),
            false,
          );
          return;
        }
        callback(null, true);
      },
    }),
  )
  async uploadImage(@UploadedFile() file: Express.Multer.File): Promise<{ url: string }> {
    if (!file) {
      throw new BadRequestException('No file uploaded');
    }

    let webp: Buffer;
    try {
      webp = await convertImageBufferToWebp(file.buffer);
    } catch {
      throw new BadRequestException('Could not process image file');
    }

    const filename = `${randomUUID()}.webp`;
    await writeFile(join(UPLOADS_DIR, filename), webp);
    return { url: `/uploads/${filename}` };
  }

  @Post('file')
  @UseInterceptors(
    FileInterceptor('file', {
      storage: diskStorage({
        destination: UPLOADS_DIR,
        filename: (_req, file, callback) => callback(null, randomFilename(file.originalname)),
      }),
      limits: { fileSize: MAX_FILE_SIZE_BYTES },
      fileFilter: (_req, file, callback) => {
        const isAllowedMime = ALLOWED_FILE_MIME_TYPES.has(file.mimetype);
        const isGpx = extname(file.originalname).toLowerCase() === '.gpx';
        if (!isAllowedMime && !isGpx) {
          callback(new BadRequestException('File type is not allowed'), false);
          return;
        }
        callback(null, true);
      },
    }),
  )
  uploadFile(@UploadedFile() file: Express.Multer.File): {
    url: string;
    name: string;
    size: number;
  } {
    if (!file) {
      throw new BadRequestException('No file uploaded');
    }
    return { url: `/uploads/${file.filename}`, name: file.originalname, size: file.size };
  }
}
