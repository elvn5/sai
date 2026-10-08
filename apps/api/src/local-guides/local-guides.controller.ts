import type { Paginated } from '@/common/listing.schemas';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { CurrentUser } from '@/user-auth/current-user.decorator';
import { UserJwtAuthGuard } from '@/user-auth/user-jwt-auth.guard';
import { LocalGuide, User } from '@app/database';
import { Body, Controller, Delete, Get, Param, Post, Put, Query, UseGuards } from '@nestjs/common';
import {
  type CreateLocalGuideDto,
  type LocalGuideListQueryDto,
  type UpdateLocalGuideDto,
  createLocalGuideSchema,
  localGuideListQuerySchema,
  updateLocalGuideSchema,
} from './local-guide.dto';
import { LocalGuidesService } from './local-guides.service';

@Controller('local-guides')
export class LocalGuidesController {
  constructor(private readonly service: LocalGuidesService) {}

  @Get()
  findPublished(
    @Query(new ZodValidationPipe(localGuideListQuerySchema)) query: LocalGuideListQueryDto,
  ): Promise<Paginated<LocalGuide>> {
    return this.service.findPublished(query);
  }

  @Get('mine')
  @UseGuards(UserJwtAuthGuard)
  // Wrapped so "no profile yet" is an explicit null rather than an empty
  // response body.
  async findMine(@CurrentUser() user: User): Promise<{ guide: LocalGuide | null }> {
    return { guide: await this.service.findMine(user.id) };
  }

  @Get(':slug')
  findBySlug(@Param('slug') slug: string): Promise<LocalGuide> {
    return this.service.findBySlug(slug);
  }

  @Post()
  @UseGuards(UserJwtAuthGuard)
  create(
    @CurrentUser() user: User,
    @Body(new ZodValidationPipe(createLocalGuideSchema)) dto: CreateLocalGuideDto,
  ): Promise<LocalGuide> {
    return this.service.create(user.id, dto);
  }

  @Put(':id')
  @UseGuards(UserJwtAuthGuard)
  update(
    @CurrentUser() user: User,
    @Param('id') id: string,
    @Body(new ZodValidationPipe(updateLocalGuideSchema)) dto: UpdateLocalGuideDto,
  ): Promise<LocalGuide> {
    return this.service.update(user.id, id, dto);
  }

  @Delete(':id')
  @UseGuards(UserJwtAuthGuard)
  remove(@CurrentUser() user: User, @Param('id') id: string): Promise<void> {
    return this.service.remove(user.id, id);
  }
}
