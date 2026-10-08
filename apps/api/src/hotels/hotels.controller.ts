import type { Paginated } from '@/common/listing.schemas';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { CurrentUser } from '@/user-auth/current-user.decorator';
import { UserJwtAuthGuard } from '@/user-auth/user-jwt-auth.guard';
import { Hotel, User } from '@app/database';
import { Body, Controller, Delete, Get, Param, Post, Put, Query, UseGuards } from '@nestjs/common';
import {
  type CreateHotelDto,
  type HotelListQueryDto,
  type UpdateHotelDto,
  createHotelSchema,
  hotelListQuerySchema,
  updateHotelSchema,
} from './hotel.dto';
import { HotelsService } from './hotels.service';

@Controller('hotels')
export class HotelsController {
  constructor(private readonly service: HotelsService) {}

  @Get()
  findPublished(
    @Query(new ZodValidationPipe(hotelListQuerySchema)) query: HotelListQueryDto,
  ): Promise<Paginated<Hotel>> {
    return this.service.findPublished(query);
  }

  @Get('mine')
  @UseGuards(UserJwtAuthGuard)
  findMine(@CurrentUser() user: User): Promise<Hotel[]> {
    return this.service.findMine(user.id);
  }

  @Get(':slug')
  findBySlug(@Param('slug') slug: string): Promise<Hotel> {
    return this.service.findBySlug(slug);
  }

  @Post()
  @UseGuards(UserJwtAuthGuard)
  create(
    @CurrentUser() user: User,
    @Body(new ZodValidationPipe(createHotelSchema)) dto: CreateHotelDto,
  ): Promise<Hotel> {
    return this.service.create(user.id, dto);
  }

  @Put(':id')
  @UseGuards(UserJwtAuthGuard)
  update(
    @CurrentUser() user: User,
    @Param('id') id: string,
    @Body(new ZodValidationPipe(updateHotelSchema)) dto: UpdateHotelDto,
  ): Promise<Hotel> {
    return this.service.update(user.id, id, dto);
  }

  @Delete(':id')
  @UseGuards(UserJwtAuthGuard)
  remove(@CurrentUser() user: User, @Param('id') id: string): Promise<void> {
    return this.service.remove(user.id, id);
  }
}
