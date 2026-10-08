import type { Paginated } from '@/common/listing.schemas';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { CurrentUser } from '@/user-auth/current-user.decorator';
import { UserJwtAuthGuard } from '@/user-auth/user-jwt-auth.guard';
import { User } from '@app/database';
import { Body, Controller, Delete, Get, Param, Post, Query, UseGuards } from '@nestjs/common';
import {
  type CreateListingPostDto,
  type ListingPostsQueryDto,
  createListingPostSchema,
  listingPostsQuerySchema,
} from './listing-post.dto';
import { ListingPostsService, type PublicListingPost } from './listing-posts.service';

@Controller('listing-posts')
export class ListingPostsController {
  constructor(private readonly service: ListingPostsService) {}

  @Get()
  findForTarget(
    @Query(new ZodValidationPipe(listingPostsQuerySchema)) query: ListingPostsQueryDto,
  ): Promise<Paginated<PublicListingPost>> {
    return this.service.findForTarget(query);
  }

  @Post()
  @UseGuards(UserJwtAuthGuard)
  create(
    @CurrentUser() user: User,
    @Body(new ZodValidationPipe(createListingPostSchema)) dto: CreateListingPostDto,
  ): Promise<PublicListingPost> {
    return this.service.create(user.id, dto);
  }

  @Delete(':id')
  @UseGuards(UserJwtAuthGuard)
  remove(@CurrentUser() user: User, @Param('id') id: string): Promise<void> {
    return this.service.remove(user.id, id);
  }
}
