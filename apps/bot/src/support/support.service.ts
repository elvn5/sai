import { SupportInfo } from '@app/database';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';

const DEFAULT_MESSAGE = 'Служба поддержки пока не настроена. Обратитесь к администратору.';

export interface SupportContact {
  message: string;
  phone: string | null;
  telegramContact: string | null;
  email: string | null;
  workingHours: string | null;
}

@Injectable()
export class SupportService {
  constructor(
    @InjectRepository(SupportInfo)
    private readonly repository: Repository<SupportInfo>,
  ) {}

  async get(): Promise<SupportContact> {
    const existing = await this.repository.find({ take: 1 });
    return (
      existing[0] ?? {
        message: DEFAULT_MESSAGE,
        phone: null,
        telegramContact: null,
        email: null,
        workingHours: null,
      }
    );
  }
}
