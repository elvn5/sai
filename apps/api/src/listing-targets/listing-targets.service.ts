import { Hotel, ListingPost, LocalGuide, Review, ReviewTargetType, Tour } from '@app/database';
import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectDataSource } from '@nestjs/typeorm';
import type { DataSource, Repository } from 'typeorm';

interface RatedListing {
  id: string;
  userId: string | null;
  isPublished: boolean;
  ratingAverage: number;
  reviewCount: number;
}

const TARGET_ENTITIES = {
  [ReviewTargetType.HOTEL]: Hotel,
  [ReviewTargetType.LOCAL_GUIDE]: LocalGuide,
  [ReviewTargetType.TOUR]: Tour,
} as const;

/**
 * The single place that knows how a polymorphic (targetType, targetId) pair
 * maps onto the hotels / local_guides / tours tables, shared by reviews and
 * owner posts.
 */
@Injectable()
export class ListingTargetsService {
  constructor(@InjectDataSource() private readonly dataSource: DataSource) {}

  async findOwnerId(type: ReviewTargetType, id: string): Promise<string | null> {
    const target = await this.repository(type).findOne({
      where: { id, isPublished: true },
      select: { id: true, userId: true },
    });
    if (!target) {
      throw new NotFoundException('Listing not found');
    }
    return target.userId;
  }

  async refreshRating(type: ReviewTargetType, id: string): Promise<void> {
    const stats = await this.dataSource
      .getRepository(Review)
      .createQueryBuilder('review')
      .select('COALESCE(AVG(review.rating), 0)', 'average')
      .addSelect('COUNT(*)', 'count')
      .where('review.targetType = :type AND review.targetId = :id', { type, id })
      .getRawOne<{ average: string; count: string }>();

    await this.repository(type).update(id, {
      ratingAverage: Math.round(Number(stats?.average ?? 0) * 100) / 100,
      reviewCount: Number(stats?.count ?? 0),
    });
  }

  // Reviews and posts reference their listing polymorphically, so there's
  // no FK cascade — listing services call this before deleting.
  async deleteContentFor(type: ReviewTargetType, id: string): Promise<void> {
    await this.dataSource.getRepository(Review).delete({ targetType: type, targetId: id });
    await this.dataSource.getRepository(ListingPost).delete({ targetType: type, targetId: id });
  }

  private repository(type: ReviewTargetType): Repository<RatedListing> {
    return this.dataSource.getRepository(
      TARGET_ENTITIES[type],
    ) as unknown as Repository<RatedListing>;
  }
}
