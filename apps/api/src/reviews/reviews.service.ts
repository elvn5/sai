import { type Paginated } from '@/common/listing.schemas';
import { type PublicAuthor, toPublicAuthor } from '@/common/public-author';
import { ListingTargetsService } from '@/listing-targets/listing-targets.service';
import { Review, type ReviewTargetType } from '@app/database';
import {
  ConflictException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateReviewDto, ReviewTargetQueryDto } from './review.dto';

export interface PublicReview {
  id: string;
  rating: number;
  text: string;
  photos: string[];
  createdAt: Date;
  author: PublicAuthor;
}

export interface ReviewSummary {
  average: number;
  count: number;
  // Number of reviews per star rating, keyed "1".."5".
  distribution: Record<string, number>;
}

export interface ReviewsPage extends Paginated<PublicReview> {
  summary: ReviewSummary;
}

@Injectable()
export class ReviewsService {
  constructor(
    @InjectRepository(Review)
    private readonly repository: Repository<Review>,
    private readonly targets: ListingTargetsService,
  ) {}

  async findForTarget(query: ReviewTargetQueryDto): Promise<ReviewsPage> {
    const where = { targetType: query.targetType, targetId: query.targetId };
    const [reviews, total] = await this.repository.findAndCount({
      where,
      relations: { user: true },
      order: { createdAt: 'DESC' },
      take: query.limit,
      skip: query.offset,
    });

    return {
      items: reviews.map(toPublicReview),
      total,
      summary: await this.summarize(query.targetType, query.targetId),
    };
  }

  async create(userId: string, dto: CreateReviewDto): Promise<PublicReview> {
    const ownerId = await this.targets.findOwnerId(dto.targetType, dto.targetId);
    if (ownerId === userId) {
      throw new ForbiddenException('You cannot review your own listing');
    }

    const existing = await this.repository.findOne({
      where: { targetType: dto.targetType, targetId: dto.targetId, userId },
    });
    if (existing) {
      throw new ConflictException('You have already reviewed this listing');
    }

    const saved = await this.repository.save(this.repository.create({ ...dto, userId }));
    await this.targets.refreshRating(dto.targetType, dto.targetId);

    const withAuthor = await this.repository.findOneOrFail({
      where: { id: saved.id },
      relations: { user: true },
    });
    return toPublicReview(withAuthor);
  }

  async remove(userId: string, id: string): Promise<void> {
    const review = await this.repository.findOne({ where: { id } });
    if (!review) {
      throw new NotFoundException('Review not found');
    }
    if (review.userId !== userId) {
      throw new ForbiddenException('You can only delete your own reviews');
    }
    await this.repository.remove(review);
    await this.targets.refreshRating(review.targetType, review.targetId);
  }

  private async summarize(type: ReviewTargetType, id: string): Promise<ReviewSummary> {
    const rows = await this.repository
      .createQueryBuilder('review')
      .select('review.rating', 'rating')
      .addSelect('COUNT(*)', 'count')
      .where('review.targetType = :type AND review.targetId = :id', { type, id })
      .groupBy('review.rating')
      .getRawMany<{ rating: number; count: string }>();

    const distribution: Record<string, number> = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0 };
    let count = 0;
    let sum = 0;
    for (const row of rows) {
      const rowCount = Number(row.count);
      distribution[String(row.rating)] = rowCount;
      count += rowCount;
      sum += Number(row.rating) * rowCount;
    }

    return {
      average: count > 0 ? Math.round((sum / count) * 100) / 100 : 0,
      count,
      distribution,
    };
  }
}

function toPublicReview(review: Review): PublicReview {
  return {
    id: review.id,
    rating: review.rating,
    text: review.text,
    photos: review.photos,
    createdAt: review.createdAt,
    author: toPublicAuthor(review.user),
  };
}
