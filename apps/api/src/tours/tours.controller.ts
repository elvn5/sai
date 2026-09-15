import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { CurrentUser } from '@/user-auth/current-user.decorator';
import { PartnerGuard } from '@/user-auth/partner.guard';
import { UserJwtAuthGuard } from '@/user-auth/user-jwt-auth.guard';
import { Locale, User } from '@app/database';
import { Body, Controller, Delete, Get, Param, Post, Put, Query, UseGuards } from '@nestjs/common';
import {
  type CreateTourDto,
  type UpdateTourDto,
  createTourSchema,
  updateTourSchema,
} from './tour.dto';
import { ToursService } from './tours.service';

@Controller('tours')
export class ToursController {
  constructor(private readonly service: ToursService) {}

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
  @UseGuards(UserJwtAuthGuard, PartnerGuard)
  create(
    @CurrentUser() user: User,
    @Body(new ZodValidationPipe(createTourSchema)) dto: CreateTourDto,
  ) {
    return this.service.create(user.id, dto);
  }

  @Put(':id')
  @UseGuards(UserJwtAuthGuard)
  update(
    @CurrentUser() user: User,
    @Param('id') id: string,
    @Body(new ZodValidationPipe(updateTourSchema)) dto: UpdateTourDto,
  ) {
    return this.service.update(user.id, id, dto);
  }

  @Delete(':id')
  @UseGuards(UserJwtAuthGuard)
  remove(@CurrentUser() user: User, @Param('id') id: string) {
    return this.service.remove(user.id, id);
  }
}
