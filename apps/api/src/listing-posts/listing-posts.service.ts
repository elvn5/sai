import type { Paginated } from '@/common/listing.schemas';
import { type PublicAuthor, toPublicAuthor } from '@/common/public-author';
import { ListingTargetsService } from '@/listing-targets/listing-targets.service';
import { ListingPost } from '@app/database';
import { ForbiddenException, Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateListingPostDto, ListingPostsQueryDto } from './listing-post.dto';

export interface PublicListingPost {
  id: string;
  text: string;
  photos: string[];
  createdAt: Date;
  author: PublicAuthor;
}

@Injectable()
export class ListingPostsService {
  constructor(
    @InjectRepository(ListingPost)
    private readonly repository: Repository<ListingPost>,
    private readonly targets: ListingTargetsService,
  ) {}

  async findForTarget(query: ListingPostsQueryDto): Promise<Paginated<PublicListingPost>> {
    const [posts, total] = await this.repository.findAndCount({
      where: { targetType: query.targetType, targetId: query.targetId },
      relations: { user: true },
      order: { createdAt: 'DESC' },
      take: query.limit,
      skip: query.offset,
    });
    return { items: posts.map(toPublicPost), total };
  }

  async create(userId: string, dto: CreateListingPostDto): Promise<PublicListingPost> {
    const ownerId = await this.targets.findOwnerId(dto.targetType, dto.targetId);
    if (ownerId !== userId) {
      throw new ForbiddenException('Only the listing owner can post here');
    }

    const saved = await this.repository.save(this.repository.create({ ...dto, userId }));
    const withAuthor = await this.repository.findOneOrFail({
      where: { id: saved.id },
      relations: { user: true },
    });
    return toPublicPost(withAuthor);
  }

  async remove(userId: string, id: string): Promise<void> {
    const post = await this.repository.findOne({ where: { id } });
    if (!post) {
      throw new NotFoundException('Post not found');
    }
    if (post.userId !== userId) {
      throw new ForbiddenException('You can only delete your own posts');
    }
    await this.repository.remove(post);
  }
}

function toPublicPost(post: ListingPost): PublicListingPost {
  return {
    id: post.id,
    text: post.text,
    photos: post.photos,
    createdAt: post.createdAt,
    author: toPublicAuthor(post.user),
  };
}
