import { BannedItem } from '@app/database';
import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateBannedItemDto, UpdateBannedItemDto } from './banned-item.dto';

@Injectable()
export class BannedItemsService {
  constructor(
    @InjectRepository(BannedItem)
    private readonly repository: Repository<BannedItem>,
  ) {}

  findAll(): Promise<BannedItem[]> {
    return this.repository.find({ order: { sortOrder: 'ASC', name: 'ASC' } });
  }

  async findOne(id: string): Promise<BannedItem> {
    const item = await this.repository.findOne({ where: { id } });
    if (!item) {
      throw new NotFoundException('Banned item not found');
    }
    return item;
  }

  create(dto: CreateBannedItemDto): Promise<BannedItem> {
    const item = this.repository.create(dto);
    return this.repository.save(item);
  }

  async update(id: string, dto: UpdateBannedItemDto): Promise<BannedItem> {
    const item = await this.findOne(id);
    Object.assign(item, dto);
    return this.repository.save(item);
  }

  async remove(id: string): Promise<void> {
    const item = await this.findOne(id);
    await this.repository.remove(item);
  }
}
