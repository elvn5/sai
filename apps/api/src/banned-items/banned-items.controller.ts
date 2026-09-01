import { JwtAuthGuard } from '@/auth/jwt-auth.guard';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { Body, Controller, Delete, Get, Param, Post, Put, UseGuards } from '@nestjs/common';
import {
  type CreateBannedItemDto,
  type UpdateBannedItemDto,
  createBannedItemSchema,
  updateBannedItemSchema,
} from './banned-item.dto';
import { BannedItemsService } from './banned-items.service';

@Controller('banned-items')
@UseGuards(JwtAuthGuard)
export class BannedItemsController {
  constructor(private readonly service: BannedItemsService) {}

  @Get()
  findAll() {
    return this.service.findAll();
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.service.findOne(id);
  }

  @Post()
  create(@Body(new ZodValidationPipe(createBannedItemSchema)) dto: CreateBannedItemDto) {
    return this.service.create(dto);
  }

  @Put(':id')
  update(
    @Param('id') id: string,
    @Body(new ZodValidationPipe(updateBannedItemSchema)) dto: UpdateBannedItemDto,
  ) {
    return this.service.update(id, dto);
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return this.service.remove(id);
  }
}
