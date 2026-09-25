import { GalleryItem } from '@app/database';
import { ForbiddenException, Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateGalleryItemDto, UpdateGalleryItemDto } from './gallery.dto';

export interface FindPublishedOptions {
  location?: string;
  search?: string;
  limit: number;
  offset: number;
}

@Injectable()
export class GalleryService {
  constructor(
    @InjectRepository(GalleryItem)
    private readonly repository: Repository<GalleryItem>,
  ) {}

  findPublished(options: FindPublishedOptions): Promise<GalleryItem[]> {
    const qb = this.repository
      .createQueryBuilder('item')
      .where('item.isPublished = :isPublished', { isPublished: true })
      .orderBy('item.sortOrder', 'ASC')
      .addOrderBy('item.createdAt', 'DESC')
      .take(options.limit)
      .skip(options.offset);

    if (options.location) {
      qb.andWhere('item.location = :location', { location: options.location });
    }
    if (options.search) {
      qb.andWhere('(item.caption ILIKE :search OR item.location ILIKE :search)', {
        search: `%${options.search}%`,
      });
    }

    return qb.getMany();
  }

  async findLocations(): Promise<string[]> {
    const rows = await this.repository
      .createQueryBuilder('item')
      .select('DISTINCT item.location', 'location')
      .where('item.isPublished = :isPublished', { isPublished: true })
      .andWhere('item.location IS NOT NULL')
      .orderBy('item.location', 'ASC')
      .getRawMany<{ location: string }>();
    return rows.map((row) => row.location);
  }

  findMine(userId: string): Promise<GalleryItem[]> {
    return this.repository.find({ where: { userId }, order: { createdAt: 'DESC' } });
  }

  async findOne(id: string): Promise<GalleryItem> {
    const item = await this.repository.findOne({ where: { id } });
    if (!item) {
      throw new NotFoundException('Gallery item not found');
    }
    return item;
  }

  create(userId: string, dto: CreateGalleryItemDto): Promise<GalleryItem> {
    const item = this.repository.create({ ...dto, userId, isPublished: true, sortOrder: 0 });
    return this.repository.save(item);
  }

  async update(userId: string, id: string, dto: UpdateGalleryItemDto): Promise<GalleryItem> {
    const item = await this.findOwned(userId, id);
    Object.assign(item, dto);
    return this.repository.save(item);
  }

  async remove(userId: string, id: string): Promise<void> {
    const item = await this.findOwned(userId, id);
    await this.repository.remove(item);
  }

  private async findOwned(userId: string, id: string): Promise<GalleryItem> {
    const item = await this.findOne(id);
    if (item.userId !== userId) {
      throw new ForbiddenException('You can only manage your own gallery items');
    }
    return item;
  }
}
