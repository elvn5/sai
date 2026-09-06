import { SupportInfo } from '@app/database';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { UpdateSupportInfoDto } from './support.dto';

const DEFAULT_MESSAGE = 'Служба поддержки пока не настроена. Обратитесь к администратору.';

@Injectable()
export class SupportService {
  constructor(
    @InjectRepository(SupportInfo)
    private readonly repository: Repository<SupportInfo>,
  ) {}

  async get(): Promise<SupportInfo> {
    const existing = await this.repository.find({ take: 1 });
    if (existing[0]) {
      return existing[0];
    }
    const created = this.repository.create({ message: DEFAULT_MESSAGE });
    return this.repository.save(created);
  }

  async update(dto: UpdateSupportInfoDto): Promise<SupportInfo> {
    const current = await this.get();
    Object.assign(current, dto);
    return this.repository.save(current);
  }
}
