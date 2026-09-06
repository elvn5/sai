import { AuthModule } from '@/auth/auth.module';
import { Post } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { PostsController } from './posts.controller';
import { PostsService } from './posts.service';

@Module({
  imports: [TypeOrmModule.forFeature([Post]), AuthModule],
  controllers: [PostsController],
  providers: [PostsService],
})
export class PostsModule {}
