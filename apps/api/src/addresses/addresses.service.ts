import { Address } from '@app/database';
import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateAddressDto, UpdateAddressDto } from './address.dto';

@Injectable()
export class AddressesService {
  constructor(
    @InjectRepository(Address)
    private readonly repository: Repository<Address>,
  ) {}

  findAll(): Promise<Address[]> {
    return this.repository.find({ order: { sortOrder: 'ASC', title: 'ASC' } });
  }

  async findOne(id: string): Promise<Address> {
    const item = await this.repository.findOne({ where: { id } });
    if (!item) {
      throw new NotFoundException('Address not found');
    }
    return item;
  }

  create(dto: CreateAddressDto): Promise<Address> {
    const item = this.repository.create(dto);
    return this.repository.save(item);
  }

  async update(id: string, dto: UpdateAddressDto): Promise<Address> {
    const item = await this.findOne(id);
    Object.assign(item, dto);
    return this.repository.save(item);
  }

  async remove(id: string): Promise<void> {
    const item = await this.findOne(id);
    await this.repository.remove(item);
  }
}
