import { Address } from '@app/database';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';

@Injectable()
export class AddressesService {
  constructor(
    @InjectRepository(Address)
    private readonly repository: Repository<Address>,
  ) {}

  findActive(): Promise<Address[]> {
    return this.repository.find({
      where: { isActive: true },
      order: { sortOrder: 'ASC', title: 'ASC' },
    });
  }
}
