import { User } from '@app/database';
import { Injectable, UnauthorizedException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { JwtService } from '@nestjs/jwt';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';
import type { PublicTelegramUser, TelegramWidgetLoginDto } from './telegram-auth.dto';
import { verifyTelegramInitData, verifyTelegramWidgetHash } from './telegram-verify.util';

const MAX_AUTH_AGE_SECONDS = 24 * 60 * 60;

interface TelegramProfile {
  telegramId: string;
  username?: string;
  firstName?: string;
  lastName?: string;
  avatarUrl?: string;
}

@Injectable()
export class TelegramAuthService {
  constructor(
    private readonly configService: ConfigService,
    private readonly jwtService: JwtService,
    @InjectRepository(User)
    private readonly users: Repository<User>,
  ) {}

  async loginWithWidget(
    dto: TelegramWidgetLoginDto,
  ): Promise<{ token: string; user: PublicTelegramUser }> {
    const botToken = this.getBotToken();
    if (!verifyTelegramWidgetHash(dto, botToken)) {
      throw new UnauthorizedException('Invalid Telegram signature');
    }
    this.assertFresh(dto.auth_date);

    const user = await this.findOrCreate({
      telegramId: String(dto.id),
      username: dto.username,
      firstName: dto.first_name,
      lastName: dto.last_name,
      avatarUrl: dto.photo_url,
    });
    return this.issueSession(user);
  }

  async loginWithMiniApp(initData: string): Promise<{ token: string; user: PublicTelegramUser }> {
    const botToken = this.getBotToken();
    const params = verifyTelegramInitData(initData, botToken);
    if (!params) {
      throw new UnauthorizedException('Invalid Telegram Mini App signature');
    }

    const authDate = Number(params.get('auth_date'));
    this.assertFresh(authDate);

    const userRaw = params.get('user');
    if (!userRaw) {
      throw new UnauthorizedException('Missing user in initData');
    }

    let parsed: {
      id: number;
      first_name: string;
      last_name?: string;
      username?: string;
      photo_url?: string;
    };
    try {
      parsed = JSON.parse(userRaw);
    } catch {
      throw new UnauthorizedException('Malformed user in initData');
    }

    const user = await this.findOrCreate({
      telegramId: String(parsed.id),
      username: parsed.username,
      firstName: parsed.first_name,
      lastName: parsed.last_name,
      avatarUrl: parsed.photo_url,
    });
    return this.issueSession(user);
  }

  async getPublicUser(userId: string): Promise<PublicTelegramUser> {
    const user = await this.users.findOne({ where: { id: userId } });
    if (!user) {
      throw new UnauthorizedException('User not found');
    }
    return this.toPublic(user);
  }

  private getBotToken(): string {
    const token = this.configService.get<string>('BOT_TOKEN');
    if (!token) {
      throw new UnauthorizedException('Bot token is not configured');
    }
    return token;
  }

  private assertFresh(authDateSeconds: number): void {
    const ageSeconds = Date.now() / 1000 - authDateSeconds;
    if (!Number.isFinite(ageSeconds) || ageSeconds > MAX_AUTH_AGE_SECONDS || ageSeconds < -60) {
      throw new UnauthorizedException('Telegram auth data has expired');
    }
  }

  private async findOrCreate(profile: TelegramProfile): Promise<User> {
    const existing = await this.users.findOne({ where: { telegramId: profile.telegramId } });
    if (existing) {
      existing.username = profile.username ?? existing.username;
      existing.firstName = profile.firstName ?? existing.firstName;
      existing.lastName = profile.lastName ?? existing.lastName;
      existing.avatarUrl = profile.avatarUrl ?? existing.avatarUrl;
      return this.users.save(existing);
    }

    const created = this.users.create({
      telegramId: profile.telegramId,
      username: profile.username ?? null,
      firstName: profile.firstName ?? null,
      lastName: profile.lastName ?? null,
      phone: null,
      avatarUrl: profile.avatarUrl ?? null,
    });
    return this.users.save(created);
  }

  private issueSession(user: User): { token: string; user: PublicTelegramUser } {
    const token = this.jwtService.sign({ sub: user.id, role: 'user' });
    return { token, user: this.toPublic(user) };
  }

  private toPublic(user: User): PublicTelegramUser {
    return {
      id: user.id,
      telegramId: user.telegramId,
      username: user.username,
      firstName: user.firstName,
      lastName: user.lastName,
      avatarUrl: user.avatarUrl,
    };
  }
}
