import { JwtAuthGuard } from '@/auth/jwt-auth.guard';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { Body, Controller, Delete, Get, Param, Post, Put, UseGuards } from '@nestjs/common';
import {
  type CreateAddressDto,
  type UpdateAddressDto,
  createAddressSchema,
  updateAddressSchema,
} from './address.dto';
import { AddressesService } from './addresses.service';

@Controller('addresses')
@UseGuards(JwtAuthGuard)
export class AddressesController {
  constructor(private readonly service: AddressesService) {}

  @Get()
  findAll() {
    return this.service.findAll();
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return this.service.findOne(id);
  }

  @Post()
  create(@Body(new ZodValidationPipe(createAddressSchema)) dto: CreateAddressDto) {
    return this.service.create(dto);
  }

  @Put(':id')
  update(
    @Param('id') id: string,
    @Body(new ZodValidationPipe(updateAddressSchema)) dto: UpdateAddressDto,
  ) {
    return this.service.update(id, dto);
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return this.service.remove(id);
  }
}
