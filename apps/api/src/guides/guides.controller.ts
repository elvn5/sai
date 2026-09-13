import { JwtAuthGuard } from '@/auth/jwt-auth.guard';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { Locale } from '@app/database';
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

  @Get('admin/all')
  @UseGuards(JwtAuthGuard)
  findAllAdmin(@Query('locale') locale?: Locale) {
    return this.service.findAllAdmin(locale);
  }

  @Get('admin/:id')
  @UseGuards(JwtAuthGuard)
  findOneAdmin(@Param('id') id: string) {
    return this.service.findOneAdmin(id);
  }

  @Get(':locale/:slug')
  findBySlug(@Param('locale') locale: Locale, @Param('slug') slug: string) {
    return this.service.findBySlug(locale, slug);
  }

  @Post()
  @UseGuards(JwtAuthGuard)
  create(@Body(new ZodValidationPipe(createGuideSchema)) dto: CreateGuideDto) {
    return this.service.create(dto);
  }

  @Put(':id')
  @UseGuards(JwtAuthGuard)
  update(
    @Param('id') id: string,
    @Body(new ZodValidationPipe(updateGuideSchema)) dto: UpdateGuideDto,
  ) {
    return this.service.update(id, dto);
  }

  @Delete(':id')
  @UseGuards(JwtAuthGuard)
  remove(@Param('id') id: string) {
    return this.service.remove(id);
  }
}
