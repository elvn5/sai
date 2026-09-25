import { AnyAuthGuard } from '@/common/any-auth.guard';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { Body, Controller, Delete, Post, UseGuards } from '@nestjs/common';
import {
  type DeleteFileDto,
  type PresignedUploadDto,
  deleteFileSchema,
  presignedUploadSchema,
} from './storage.dto';
import { type PresignedUpload, StorageService } from './storage.service';

@Controller('storage')
@UseGuards(AnyAuthGuard)
export class StorageController {
  constructor(private readonly service: StorageService) {}

  @Post('upload-url')
  generateUploadUrl(
    @Body(new ZodValidationPipe(presignedUploadSchema)) dto: PresignedUploadDto,
  ): Promise<PresignedUpload> {
    return this.service.generatePresignedUploadUrl(dto.filename, dto.contentType, dto.folder);
  }

  @Delete('file')
  async deleteFile(
    @Body(new ZodValidationPipe(deleteFileSchema)) dto: DeleteFileDto,
  ): Promise<{ success: true }> {
    await this.service.deleteFile(dto.key);
    return { success: true };
  }
}
