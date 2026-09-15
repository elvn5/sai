import { sanitizeUserContent } from '@/common/sanitize-user-content';
import { Guide, type Locale } from '@app/database';
import {
  ConflictException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateGuideDto, UpdateGuideDto } from './guide.dto';

@Injectable()
export class GuidesService {
  constructor(
    @InjectRepository(Guide)
    private readonly repository: Repository<Guide>,
  ) {}

  findPublished(locale: Locale): Promise<Guide[]> {
    return this.repository.find({
      where: { locale, isPublished: true },
      order: { publishedAt: 'DESC', createdAt: 'DESC' },
    });
  }

  async findBySlug(locale: Locale, slug: string): Promise<Guide> {
    const item = await this.repository.findOne({ where: { locale, slug, isPublished: true } });
    if (!item) {
      throw new NotFoundException('Guide not found');
    }
    return item;
  }

  findMine(userId: string): Promise<Guide[]> {
    return this.repository.find({ where: { userId }, order: { createdAt: 'DESC' } });
  }

  async create(userId: string, dto: CreateGuideDto): Promise<Guide> {
    await this.assertSlugAvailable(dto.locale, dto.slug);
    const item = this.repository.create({
      ...dto,
      content: sanitizeUserContent(dto.content),
      userId,
      isPublished: true,
      publishedAt: new Date(),
    });
    return this.repository.save(item);
  }

  async update(userId: string, id: string, dto: UpdateGuideDto): Promise<Guide> {
    const item = await this.findOwned(userId, id);
    if (dto.slug && dto.slug !== item.slug) {
      await this.assertSlugAvailable(dto.locale ?? item.locale, dto.slug, id);
    }

    Object.assign(item, dto);
    if (dto.content) {
      item.content = sanitizeUserContent(dto.content);
    }

    return this.repository.save(item);
  }

  async remove(userId: string, id: string): Promise<void> {
    const item = await this.findOwned(userId, id);
    await this.repository.remove(item);
  }

  private async findOwned(userId: string, id: string): Promise<Guide> {
    const item = await this.repository.findOne({ where: { id } });
    if (!item) {
      throw new NotFoundException('Guide not found');
    }
    if (item.userId !== userId) {
      throw new ForbiddenException('You can only manage your own guides');
    }
    return item;
  }

  private async assertSlugAvailable(
    locale: Locale,
    slug: string,
    excludeId?: string,
  ): Promise<void> {
    const existing = await this.repository.findOne({ where: { locale, slug } });
    if (existing && existing.id !== excludeId) {
      throw new ConflictException('Slug is already used for this locale');
    }
  }
}
