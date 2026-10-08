import type { Paginated } from '@/common/listing.schemas';
import { sanitizeUserContent } from '@/common/sanitize-user-content';
import { whereSimpleArrayContainsAll } from '@/common/simple-array-filter';
import { ListingTargetsService } from '@/listing-targets/listing-targets.service';
import { LocalGuide, ReviewTargetType } from '@app/database';
import {
  ConflictException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type {
  CreateLocalGuideDto,
  LocalGuideListQueryDto,
  UpdateLocalGuideDto,
} from './local-guide.dto';

@Injectable()
export class LocalGuidesService {
  constructor(
    @InjectRepository(LocalGuide)
    private readonly repository: Repository<LocalGuide>,
    private readonly targets: ListingTargetsService,
  ) {}

  async findPublished(query: LocalGuideListQueryDto): Promise<Paginated<LocalGuide>> {
    const qb = this.repository
      .createQueryBuilder('guide')
      .where('guide.isPublished = true')
      .take(query.limit)
      .skip(query.offset);

    if (query.destination) {
      whereSimpleArrayContainsAll(qb, 'guide.destinations', [query.destination], 'destination');
    }
    if (query.language) {
      whereSimpleArrayContainsAll(qb, 'guide.languages', [query.language], 'language');
    }
    if (query.specialization) {
      whereSimpleArrayContainsAll(
        qb,
        'guide.specializations',
        [query.specialization],
        'specialization',
      );
    }
    if (query.maxPrice !== undefined) {
      qb.andWhere('guide.pricePerDay <= :maxPrice', { maxPrice: query.maxPrice });
    }
    if (query.minRating !== undefined) {
      qb.andWhere('guide.ratingAverage >= :minRating', { minRating: query.minRating });
    }
    if (query.hasVehicle) {
      qb.andWhere('guide.hasVehicle = true');
    }
    if (query.search) {
      qb.andWhere('(guide.name ILIKE :search OR guide.headline ILIKE :search)', {
        search: `%${query.search}%`,
      });
    }

    switch (query.sort) {
      case 'price_asc':
        qb.orderBy('guide.pricePerDay', 'ASC', 'NULLS LAST');
        break;
      case 'price_desc':
        qb.orderBy('guide.pricePerDay', 'DESC', 'NULLS LAST');
        break;
      case 'experience':
        qb.orderBy('guide.experienceYears', 'DESC', 'NULLS LAST');
        break;
      case 'newest':
        qb.orderBy('guide.createdAt', 'DESC');
        break;
      default:
        qb.orderBy('guide.ratingAverage', 'DESC').addOrderBy('guide.reviewCount', 'DESC');
    }
    qb.addOrderBy('guide.createdAt', 'DESC');

    const [items, total] = await qb.getManyAndCount();
    return { items, total };
  }

  async findBySlug(slug: string): Promise<LocalGuide> {
    const guide = await this.repository.findOne({ where: { slug, isPublished: true } });
    if (!guide) {
      throw new NotFoundException('Guide not found');
    }
    return guide;
  }

  // Each user has at most one guide profile.
  findMine(userId: string): Promise<LocalGuide | null> {
    return this.repository.findOne({ where: { userId } });
  }

  async create(userId: string, dto: CreateLocalGuideDto): Promise<LocalGuide> {
    if (await this.findMine(userId)) {
      throw new ConflictException('You already have a guide profile');
    }
    await this.assertSlugAvailable(dto.slug);

    const guide = this.repository.create({
      ...dto,
      bio: sanitizeUserContent(dto.bio),
      userId,
      isPublished: true,
    });
    return this.repository.save(guide);
  }

  async update(userId: string, id: string, dto: UpdateLocalGuideDto): Promise<LocalGuide> {
    const guide = await this.findOwned(userId, id);
    if (dto.slug && dto.slug !== guide.slug) {
      await this.assertSlugAvailable(dto.slug, id);
    }

    Object.assign(guide, dto);
    if (dto.bio) {
      guide.bio = sanitizeUserContent(dto.bio);
    }
    return this.repository.save(guide);
  }

  async remove(userId: string, id: string): Promise<void> {
    const guide = await this.findOwned(userId, id);
    await this.targets.deleteContentFor(ReviewTargetType.LOCAL_GUIDE, guide.id);
    await this.repository.remove(guide);
  }

  private async findOwned(userId: string, id: string): Promise<LocalGuide> {
    const guide = await this.repository.findOne({ where: { id } });
    if (!guide) {
      throw new NotFoundException('Guide not found');
    }
    if (guide.userId !== userId) {
      throw new ForbiddenException('You can only manage your own guide profile');
    }
    return guide;
  }

  private async assertSlugAvailable(slug: string, excludeId?: string): Promise<void> {
    const existing = await this.repository.findOne({ where: { slug } });
    if (existing && existing.id !== excludeId) {
      throw new ConflictException('Slug is already taken');
    }
  }
}
