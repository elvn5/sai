import { ListingTargetsModule } from '@/listing-targets/listing-targets.module';
import { UserAuthModule } from '@/user-auth/user-auth.module';
import { ListingPost } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { ListingPostsController } from './listing-posts.controller';
import { ListingPostsService } from './listing-posts.service';

@Module({
  imports: [TypeOrmModule.forFeature([ListingPost]), UserAuthModule, ListingTargetsModule],
  controllers: [ListingPostsController],
  providers: [ListingPostsService],
})
export class ListingPostsModule {}
