import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { CurrentUser } from '@/user-auth/current-user.decorator';
import { UserJwtAuthGuard } from '@/user-auth/user-jwt-auth.guard';
import { Locale, User } from '@app/database';
import { Body, Controller, Delete, Get, Param, Post, Put, Query, UseGuards } from '@nestjs/common';
import {
  type CreateGuideDto,
  type UpdateGuideDto,
  createGuideSchema,
  updateGuideSchema,
} from './guide.dto';
import { GuidesService } from './guides.service';

@Controller('guides')
export class GuidesController {
  constructor(private readonly service: GuidesService) {}

  @Get()
  findPublished(@Query('locale') locale: Locale) {
    return this.service.findPublished(locale);
  }

  @Get('mine')
  @UseGuards(UserJwtAuthGuard)
  findMine(@CurrentUser() user: User) {
    return this.service.findMine(user.id);
  }

  @Get(':locale/:slug')
  findBySlug(@Param('locale') locale: Locale, @Param('slug') slug: string) {
    return this.service.findBySlug(locale, slug);
  }

  @Post()
  @UseGuards(UserJwtAuthGuard)
  create(
    @CurrentUser() user: User,
    @Body(new ZodValidationPipe(createGuideSchema)) dto: CreateGuideDto,
  ) {
    return this.service.create(user.id, dto);
  }

  @Put(':id')
  @UseGuards(UserJwtAuthGuard)
  update(
    @CurrentUser() user: User,
    @Param('id') id: string,
    @Body(new ZodValidationPipe(updateGuideSchema)) dto: UpdateGuideDto,
  ) {
    return this.service.update(user.id, id, dto);
  }

  @Delete(':id')
  @UseGuards(UserJwtAuthGuard)
  remove(@CurrentUser() user: User, @Param('id') id: string) {
    return this.service.remove(user.id, id);
  }
}
