import { type Locale, PageSeo, type SeoPage } from '@app/database';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { UpsertPageSeoDto } from './page-seo.dto';

@Injectable()
export class PageSeoService {
  constructor(
    @InjectRepository(PageSeo)
    private readonly repository: Repository<PageSeo>,
  ) {}

  findOne(page: SeoPage, locale: Locale): Promise<PageSeo | null> {
    return this.repository.findOne({ where: { page, locale } });
  }

  findAllAdmin(): Promise<PageSeo[]> {
    return this.repository.find();
  }

  async upsert(page: SeoPage, locale: Locale, dto: UpsertPageSeoDto): Promise<PageSeo> {
    let item = await this.repository.findOne({ where: { page, locale } });
    if (!item) {
      item = this.repository.create({ page, locale });
    }
    Object.assign(item, dto);
    return this.repository.save(item);
  }
}
