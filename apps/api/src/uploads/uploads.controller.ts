import { randomUUID } from 'node:crypto';
import { extname } from 'node:path';
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
import { diskStorage } from 'multer';
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
      storage: diskStorage({
        destination: UPLOADS_DIR,
        filename: (_req, file, callback) => callback(null, randomFilename(file.originalname)),
      }),
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
  uploadImage(@UploadedFile() file: Express.Multer.File): { url: string } {
    if (!file) {
      throw new BadRequestException('No file uploaded');
    }
    return { url: `/uploads/${file.filename}` };
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
