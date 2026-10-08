import { sanitizeUserContent } from '@/common/sanitize-user-content';
import { ListingTargetsService } from '@/listing-targets/listing-targets.service';
import { type Locale, ReviewTargetType, Tour } from '@app/database';
import {
  ConflictException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateTourDto, TourListQueryDto, UpdateTourDto } from './tour.dto';

@Injectable()
export class ToursService {
  constructor(
    @InjectRepository(Tour)
    private readonly repository: Repository<Tour>,
    private readonly targets: ListingTargetsService,
  ) {}

  findPublished(query: TourListQueryDto): Promise<Tour[]> {
    return this.repository.find({
      where: {
        locale: query.locale,
        isPublished: true,
        ...(query.destination ? { destination: query.destination } : {}),
      },
      order: { publishedAt: 'DESC', createdAt: 'DESC' },
    });
  }

  async findBySlug(locale: Locale, slug: string): Promise<Tour> {
    const item = await this.repository.findOne({ where: { locale, slug, isPublished: true } });
    if (!item) {
      throw new NotFoundException('Tour not found');
    }
    return item;
  }

  findMine(userId: string): Promise<Tour[]> {
    return this.repository.find({ where: { userId }, order: { createdAt: 'DESC' } });
  }

  async create(userId: string, dto: CreateTourDto): Promise<Tour> {
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

  async update(userId: string, id: string, dto: UpdateTourDto): Promise<Tour> {
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
    await this.targets.deleteContentFor(ReviewTargetType.TOUR, item.id);
    await this.repository.remove(item);
  }

  private async findOwned(userId: string, id: string): Promise<Tour> {
    const item = await this.repository.findOne({ where: { id } });
    if (!item) {
      throw new NotFoundException('Tour not found');
    }
    if (item.userId !== userId) {
      throw new ForbiddenException('You can only manage your own tours');
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
