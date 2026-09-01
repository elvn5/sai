import { Parcel } from '@app/database';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';

@Injectable()
export class ParcelsService {
  constructor(
    @InjectRepository(Parcel)
    private readonly repository: Repository<Parcel>,
  ) {}

  findByUserId(userId: string): Promise<Parcel[]> {
    return this.repository.find({ where: { userId }, order: { createdAt: 'DESC' } });
  }
}
