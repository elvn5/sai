import { Parcel } from '@app/database';
import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { CreateParcelDto, UpdateParcelDto } from './parcel.dto';

@Injectable()
export class ParcelsService {
  constructor(
    @InjectRepository(Parcel)
    private readonly repository: Repository<Parcel>,
  ) {}

  findAll(): Promise<Parcel[]> {
    return this.repository.find({ relations: ['user'], order: { createdAt: 'DESC' } });
  }

  async findOne(id: string): Promise<Parcel> {
    const parcel = await this.repository.findOne({ where: { id }, relations: ['user'] });
    if (!parcel) {
      throw new NotFoundException('Parcel not found');
    }
    return parcel;
  }

  create(dto: CreateParcelDto): Promise<Parcel> {
    const parcel = this.repository.create(dto);
    return this.repository.save(parcel);
  }

  async update(id: string, dto: UpdateParcelDto): Promise<Parcel> {
    const parcel = await this.findOne(id);
    Object.assign(parcel, dto);
    return this.repository.save(parcel);
  }

  async remove(id: string): Promise<void> {
    const parcel = await this.findOne(id);
    await this.repository.remove(parcel);
  }
}
