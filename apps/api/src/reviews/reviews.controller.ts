import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { CurrentUser } from '@/user-auth/current-user.decorator';
import { UserJwtAuthGuard } from '@/user-auth/user-jwt-auth.guard';
import { User } from '@app/database';
import { Body, Controller, Delete, Get, Param, Post, Query, UseGuards } from '@nestjs/common';
import {
  type CreateReviewDto,
  type ReviewTargetQueryDto,
  createReviewSchema,
  reviewTargetQuerySchema,
} from './review.dto';
import { type PublicReview, type ReviewsPage, ReviewsService } from './reviews.service';

@Controller('reviews')
export class ReviewsController {
  constructor(private readonly service: ReviewsService) {}

  @Get()
  findForTarget(
    @Query(new ZodValidationPipe(reviewTargetQuerySchema)) query: ReviewTargetQueryDto,
  ): Promise<ReviewsPage> {
    return this.service.findForTarget(query);
  }

  @Post()
  @UseGuards(UserJwtAuthGuard)
  create(
    @CurrentUser() user: User,
    @Body(new ZodValidationPipe(createReviewSchema)) dto: CreateReviewDto,
  ): Promise<PublicReview> {
    return this.service.create(user.id, dto);
  }

  @Delete(':id')
  @UseGuards(UserJwtAuthGuard)
  remove(@CurrentUser() user: User, @Param('id') id: string): Promise<void> {
    return this.service.remove(user.id, id);
  }
}
