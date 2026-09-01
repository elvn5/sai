import { BannedItem } from '@app/database';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';

@Injectable()
export class BannedItemsService {
  constructor(
    @InjectRepository(BannedItem)
    private readonly repository: Repository<BannedItem>,
  ) {}

  findActive(): Promise<BannedItem[]> {
    return this.repository.find({
      where: { isActive: true },
      order: { sortOrder: 'ASC', name: 'ASC' },
    });
  }
}
