import { StorageService } from '@/storage/storage.service';
import { GalleryItem } from '@app/database';
import { ForbiddenException, Injectable, Logger, NotFoundException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
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
  private readonly logger = new Logger(GalleryService.name);

  constructor(
    @InjectRepository(GalleryItem)
    private readonly repository: Repository<GalleryItem>,
    private readonly storageService: StorageService,
    private readonly configService: ConfigService,
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
    await this.tryDeleteR2Object(item.imageUrl);
  }

  // Photos uploaded directly to R2 store the full public URL in imageUrl;
  // legacy photos on local disk (served from /uploads/...) are left alone,
  // matching the pre-R2 behavior of never cleaning those up either.
  private async tryDeleteR2Object(imageUrl: string): Promise<void> {
    const publicUrl = this.configService.get<string>('R2_PUBLIC_URL');
    if (!publicUrl || !imageUrl.startsWith(`${publicUrl}/`)) return;

    const key = imageUrl.slice(publicUrl.length + 1);
    try {
      await this.storageService.deleteFile(key);
    } catch (error) {
      this.logger.error(`Failed to clean up R2 object "${key}" after gallery item removal`, error);
    }
  }

  private async findOwned(userId: string, id: string): Promise<GalleryItem> {
    const item = await this.findOne(id);
    if (item.userId !== userId) {
      throw new ForbiddenException('You can only manage your own gallery items');
    }
    return item;
  }
}
