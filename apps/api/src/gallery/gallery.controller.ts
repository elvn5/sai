import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { CurrentUser } from '@/user-auth/current-user.decorator';
import { UserJwtAuthGuard } from '@/user-auth/user-jwt-auth.guard';
import { GalleryRegion, User } from '@app/database';
import { Body, Controller, Delete, Get, Param, Post, Put, Query, UseGuards } from '@nestjs/common';
import {
  type CreateGalleryItemDto,
  type UpdateGalleryItemDto,
  createGalleryItemSchema,
  updateGalleryItemSchema,
} from './gallery.dto';
import { GalleryService } from './gallery.service';

@Controller('gallery')
export class GalleryController {
  constructor(private readonly service: GalleryService) {}

  @Get()
  findPublished(@Query('region') region?: string) {
    const validRegion = Object.values(GalleryRegion).includes(region as GalleryRegion)
      ? (region as GalleryRegion)
      : undefined;
    return this.service.findPublished(validRegion);
  }

  @Get('mine')
  @UseGuards(UserJwtAuthGuard)
  findMine(@CurrentUser() user: User) {
    return this.service.findMine(user.id);
  }

  @Post()
  @UseGuards(UserJwtAuthGuard)
  create(
    @CurrentUser() user: User,
    @Body(new ZodValidationPipe(createGalleryItemSchema)) dto: CreateGalleryItemDto,
  ) {
    return this.service.create(user.id, dto);
  }

  @Put(':id')
  @UseGuards(UserJwtAuthGuard)
  update(
    @CurrentUser() user: User,
    @Param('id') id: string,
    @Body(new ZodValidationPipe(updateGalleryItemSchema)) dto: UpdateGalleryItemDto,
  ) {
    return this.service.update(user.id, id, dto);
  }

  @Delete(':id')
  @UseGuards(UserJwtAuthGuard)
  remove(@CurrentUser() user: User, @Param('id') id: string) {
    return this.service.remove(user.id, id);
  }
}
