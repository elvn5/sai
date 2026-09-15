import { User } from '@app/database';
import { Injectable } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import { hashVerificationToken } from './verification-token';

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

  // Completes a phone-registration verification started on the website —
  // the site never has a Telegram chat to message directly, so it hands the
  // user a deep link instead; opening it here is what actually confirms
  // them. Returns null for an unknown, already-verified, or expired token
  // (the caller doesn't need to distinguish those in the reply).
  async verifyPendingByToken(token: string): Promise<User | null> {
    const hash = hashVerificationToken(token);
    const user = await this.repository.findOne({ where: { verificationCodeHash: hash } });
    if (
      !user ||
      user.isVerified ||
      !user.verificationCodeExpiresAt ||
      user.verificationCodeExpiresAt.getTime() < Date.now()
    ) {
      return null;
    }

    user.isVerified = true;
    user.verificationCodeHash = null;
    user.verificationCodeExpiresAt = null;
    user.verificationCodeSentAt = null;
    user.verificationAttempts = 0;
    return this.repository.save(user);
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
