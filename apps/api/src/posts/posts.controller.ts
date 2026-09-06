import { JwtAuthGuard } from '@/auth/jwt-auth.guard';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { Body, Controller, Delete, Get, Param, Post, Put, Query, UseGuards } from '@nestjs/common';
import {
  type CreatePostDto,
  type UpdatePostDto,
  createPostSchema,
  updatePostSchema,
} from './post.dto';
import { PostsService } from './posts.service';

@Controller('posts')
export class PostsController {
  constructor(private readonly service: PostsService) {}

  // -------- Публичные (без авторизации, только опубликованные) --------

  @Get()
  findPublished(@Query('language') language?: string) {
    return this.service.findPublished(language);
  }

  @Get('slug/:slug')
  findPublishedBySlug(@Param('slug') slug: string) {
    return this.service.findPublishedBySlug(slug);
  }

  // -------- Админ (JWT-кука) --------

  @Get('admin')
  @UseGuards(JwtAuthGuard)
  findAll() {
    return this.service.findAll();
  }

  @Get('admin/:id')
  @UseGuards(JwtAuthGuard)
  findOne(@Param('id') id: string) {
    return this.service.findOne(id);
  }

  @Post()
  @UseGuards(JwtAuthGuard)
  create(@Body(new ZodValidationPipe(createPostSchema)) dto: CreatePostDto) {
    return this.service.create(dto);
  }

  @Put(':id')
  @UseGuards(JwtAuthGuard)
  update(
    @Param('id') id: string,
    @Body(new ZodValidationPipe(updatePostSchema)) dto: UpdatePostDto,
  ) {
    return this.service.update(id, dto);
  }

  @Delete(':id')
  @UseGuards(JwtAuthGuard)
  remove(@Param('id') id: string) {
    return this.service.remove(id);
  }
}
