import { BlogPost, type Locale } from '@app/database';
import { ConflictException, Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateBlogPostDto, UpdateBlogPostDto } from './blog.dto';

@Injectable()
export class BlogService {
  constructor(
    @InjectRepository(BlogPost)
    private readonly repository: Repository<BlogPost>,
  ) {}

  findPublished(locale: Locale): Promise<BlogPost[]> {
    return this.repository.find({
      where: { locale, isPublished: true },
      order: { publishedAt: 'DESC', createdAt: 'DESC' },
    });
  }

  async findBySlug(locale: Locale, slug: string): Promise<BlogPost> {
    const item = await this.repository.findOne({ where: { locale, slug, isPublished: true } });
    if (!item) {
      throw new NotFoundException('Post not found');
    }
    return item;
  }

  findAllAdmin(locale?: Locale): Promise<BlogPost[]> {
    return this.repository.find({
      where: locale ? { locale } : {},
      order: { updatedAt: 'DESC' },
    });
  }

  async findOneAdmin(id: string): Promise<BlogPost> {
    const item = await this.repository.findOne({ where: { id } });
    if (!item) {
      throw new NotFoundException('Post not found');
    }
    return item;
  }

  async create(dto: CreateBlogPostDto): Promise<BlogPost> {
    await this.assertSlugAvailable(dto.locale, dto.slug);
    const item = this.repository.create({
      ...dto,
      publishedAt: dto.isPublished ? new Date() : null,
    });
    return this.repository.save(item);
  }

  async update(id: string, dto: UpdateBlogPostDto): Promise<BlogPost> {
    const item = await this.findOneAdmin(id);
    if (dto.slug && dto.slug !== item.slug) {
      await this.assertSlugAvailable(dto.locale ?? item.locale, dto.slug, id);
    }

    const wasPublished = item.isPublished;
    Object.assign(item, dto);

    if (!wasPublished && item.isPublished) {
      item.publishedAt = new Date();
    }
    if (!item.isPublished) {
      item.publishedAt = null;
    }

    return this.repository.save(item);
  }

  async remove(id: string): Promise<void> {
    const item = await this.findOneAdmin(id);
    await this.repository.remove(item);
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
