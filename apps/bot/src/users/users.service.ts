import {
  type LinkTelegramResult,
  type TelegramAccount,
  UsersGrpcClient,
} from '@/users-grpc/users-grpc.client';
import { Injectable } from '@nestjs/common';

export interface TelegramProfile {
  telegramId: string;
  username?: string;
  firstName?: string;
  lastName?: string;
}

// Account operations the bot needs, delegated to the users service.
@Injectable()
export class UsersService {
  constructor(private readonly users: UsersGrpcClient) {}

  findOrCreate(profile: TelegramProfile): Promise<TelegramAccount> {
    return this.users.upsertTelegramUser(profile);
  }

  // Completes a phone registration started on the website — the site hands
  // the user a t.me deep link; opening it here is what confirms them.
  // Returns null for an unknown, already-verified, or expired token.
  verifyPendingByToken(token: string): Promise<TelegramAccount | null> {
    return this.users.confirmTelegramVerification(token);
  }

  // Links this Telegram user to the site account that issued the token
  // (the site's "Link Telegram" deep link).
  linkTelegram(
    token: string,
    from: { id: number; username?: string },
    moveFromOtherAccount = false,
  ): Promise<LinkTelegramResult> {
    return this.users.linkTelegram({
      token,
      telegramId: String(from.id),
      username: from.username,
      moveFromOtherAccount,
    });
  }

  // Throws PhoneTakenError if another account already uses the number.
  updatePhone(userId: string, phone: string): Promise<TelegramAccount> {
    return this.users.updateProfile({ userId, phone });
  }

  updateFirstName(userId: string, firstName: string): Promise<TelegramAccount> {
    return this.users.updateProfile({ userId, firstName });
  }
}
