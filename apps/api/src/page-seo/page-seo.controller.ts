import { JwtAuthGuard } from '@/auth/jwt-auth.guard';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { type Locale, type SeoPage } from '@app/database';
import { Body, Controller, Get, Param, Put, UseGuards } from '@nestjs/common';
import { type UpsertPageSeoDto, upsertPageSeoSchema } from './page-seo.dto';
import { PageSeoService } from './page-seo.service';

@Controller('page-seo')
export class PageSeoController {
  constructor(private readonly service: PageSeoService) {}

  @Get('admin/all')
  @UseGuards(JwtAuthGuard)
  findAllAdmin() {
    return this.service.findAllAdmin();
  }

  @Get(':page/:locale')
  findOne(@Param('page') page: SeoPage, @Param('locale') locale: Locale) {
    return this.service.findOne(page, locale);
  }

  @Put(':page/:locale')
  @UseGuards(JwtAuthGuard)
  upsert(
    @Param('page') page: SeoPage,
    @Param('locale') locale: Locale,
    @Body(new ZodValidationPipe(upsertPageSeoSchema)) dto: UpsertPageSeoDto,
  ) {
    return this.service.upsert(page, locale, dto);
  }
}
