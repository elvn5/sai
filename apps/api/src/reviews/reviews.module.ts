import { ListingTargetsModule } from '@/listing-targets/listing-targets.module';
import { UserAuthModule } from '@/user-auth/user-auth.module';
import { Review } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { ReviewsController } from './reviews.controller';
import { ReviewsService } from './reviews.service';

@Module({
  imports: [TypeOrmModule.forFeature([Review]), UserAuthModule, ListingTargetsModule],
  controllers: [ReviewsController],
  providers: [ReviewsService],
})
export class ReviewsModule {}
