import { User } from '@app/database';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';

export interface TelegramProfile {
  telegramId: string;
  username?: string;
  firstName?: string;
  lastName?: string;
}

@Injectable()
export class UsersService {
  constructor(
    @InjectRepository(User)
    private readonly repository: Repository<User>,
  ) {}

  async findOrCreate(profile: TelegramProfile): Promise<User> {
    const existing = await this.repository.findOne({ where: { telegramId: profile.telegramId } });
    if (existing) {
      existing.username = profile.username ?? existing.username;
      existing.firstName = profile.firstName ?? existing.firstName;
      existing.lastName = profile.lastName ?? existing.lastName;
      return this.repository.save(existing);
    }

    const created = this.repository.create({
      telegramId: profile.telegramId,
      username: profile.username ?? null,
      firstName: profile.firstName ?? null,
      lastName: profile.lastName ?? null,
      phone: null,
    });
    return this.repository.save(created);
  }

  findByTelegramId(telegramId: string): Promise<User | null> {
    return this.repository.findOne({ where: { telegramId } });
  }

  async updatePhone(userId: string, phone: string): Promise<User> {
    return this.updateAndReturn(userId, { phone });
  }

  async updateFirstName(userId: string, firstName: string): Promise<User> {
    return this.updateAndReturn(userId, { firstName });
  }

  private async updateAndReturn(
    userId: string,
    patch: Partial<Pick<User, 'phone' | 'firstName'>>,
  ): Promise<User> {
    await this.repository.update({ id: userId }, patch);
    const updated = await this.repository.findOne({ where: { id: userId } });
    if (!updated) {
      throw new Error(`User ${userId} not found after update`);
    }
    return updated;
  }
}
