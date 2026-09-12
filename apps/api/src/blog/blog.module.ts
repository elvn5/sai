import { AuthModule } from '@/auth/auth.module';
import { BlogPost } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { BlogController } from './blog.controller';
import { BlogService } from './blog.service';

@Module({
  imports: [TypeOrmModule.forFeature([BlogPost]), AuthModule],
  controllers: [BlogController],
  providers: [BlogService],
})
export class BlogModule {}
