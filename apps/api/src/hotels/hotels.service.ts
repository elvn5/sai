import type { Paginated } from '@/common/listing.schemas';
import { sanitizeUserContent } from '@/common/sanitize-user-content';
import { whereSimpleArrayContainsAll } from '@/common/simple-array-filter';
import { ListingTargetsService } from '@/listing-targets/listing-targets.service';
import { CancellationPolicy, Hotel, ReviewTargetType } from '@app/database';
import {
  ConflictException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateHotelDto, HotelListQueryDto, UpdateHotelDto } from './hotel.dto';

@Injectable()
export class HotelsService {
  constructor(
    @InjectRepository(Hotel)
    private readonly repository: Repository<Hotel>,
    private readonly targets: ListingTargetsService,
  ) {}

  async findPublished(query: HotelListQueryDto): Promise<Paginated<Hotel>> {
    const qb = this.repository
      .createQueryBuilder('hotel')
      .where('hotel.isPublished = true')
      .take(query.limit)
      .skip(query.offset);

    if (query.destination) {
      qb.andWhere('hotel.destination = :destination', { destination: query.destination });
    }
    if (query.type) {
      qb.andWhere('hotel.type = :type', { type: query.type });
    }
    if (query.maxPrice !== undefined) {
      qb.andWhere('hotel.priceFrom <= :maxPrice', { maxPrice: query.maxPrice });
    }
    if (query.minRating !== undefined) {
      qb.andWhere('hotel.ratingAverage >= :minRating', { minRating: query.minRating });
    }
    if (query.freeCancellation) {
      qb.andWhere('hotel.cancellationPolicy = :free', { free: CancellationPolicy.FREE });
    }
    if (query.search) {
      qb.andWhere('(hotel.name ILIKE :search OR hotel.address ILIKE :search)', {
        search: `%${query.search}%`,
      });
    }
    whereSimpleArrayContainsAll(qb, 'hotel.amenities', query.amenities, 'amenity');

    switch (query.sort) {
      case 'price_asc':
        qb.orderBy('hotel.priceFrom', 'ASC', 'NULLS LAST');
        break;
      case 'price_desc':
        qb.orderBy('hotel.priceFrom', 'DESC', 'NULLS LAST');
        break;
      case 'newest':
        qb.orderBy('hotel.createdAt', 'DESC');
        break;
      default:
        qb.orderBy('hotel.ratingAverage', 'DESC').addOrderBy('hotel.reviewCount', 'DESC');
    }
    qb.addOrderBy('hotel.createdAt', 'DESC');

    const [items, total] = await qb.getManyAndCount();
    return { items, total };
  }

  async findBySlug(slug: string): Promise<Hotel> {
    const hotel = await this.repository.findOne({ where: { slug, isPublished: true } });
    if (!hotel) {
      throw new NotFoundException('Hotel not found');
    }
    return hotel;
  }

  findMine(userId: string): Promise<Hotel[]> {
    return this.repository.find({ where: { userId }, order: { createdAt: 'DESC' } });
  }

  async create(userId: string, dto: CreateHotelDto): Promise<Hotel> {
    await this.assertSlugAvailable(dto.slug);
    const hotel = this.repository.create({
      ...dto,
      description: sanitizeUserContent(dto.description),
      userId,
      isPublished: true,
    });
    return this.repository.save(hotel);
  }

  async update(userId: string, id: string, dto: UpdateHotelDto): Promise<Hotel> {
    const hotel = await this.findOwned(userId, id);
    if (dto.slug && dto.slug !== hotel.slug) {
      await this.assertSlugAvailable(dto.slug, id);
    }

    Object.assign(hotel, dto);
    if (dto.description) {
      hotel.description = sanitizeUserContent(dto.description);
    }
    return this.repository.save(hotel);
  }

  async remove(userId: string, id: string): Promise<void> {
    const hotel = await this.findOwned(userId, id);
    await this.targets.deleteContentFor(ReviewTargetType.HOTEL, hotel.id);
    await this.repository.remove(hotel);
  }

  private async findOwned(userId: string, id: string): Promise<Hotel> {
    const hotel = await this.repository.findOne({ where: { id } });
    if (!hotel) {
      throw new NotFoundException('Hotel not found');
    }
    if (hotel.userId !== userId) {
      throw new ForbiddenException('You can only manage your own hotels');
    }
    return hotel;
  }

  private async assertSlugAvailable(slug: string, excludeId?: string): Promise<void> {
    const existing = await this.repository.findOne({ where: { slug } });
    if (existing && existing.id !== excludeId) {
      throw new ConflictException('Slug is already taken');
    }
  }
}
