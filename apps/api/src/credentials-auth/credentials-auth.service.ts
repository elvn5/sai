import { EmailService } from '@/notifications/email.service';
import { type PublicUser, toPublicUser } from '@/user-auth/public-user';
import { User } from '@app/database';
import {
  BadRequestException,
  ConflictException,
  HttpException,
  HttpStatus,
  Injectable,
  NotFoundException,
  UnauthorizedException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { InjectRepository } from '@nestjs/typeorm';
import * as bcrypt from 'bcryptjs';
import type { Repository } from 'typeorm';
import type {
  LoginDto,
  RegisterDto,
  ResendCodeDto,
  VerificationStatusDto,
  VerifyCodeDto,
} from './credentials-auth.dto';
import {
  VERIFICATION_CODE_TTL_MS,
  VERIFICATION_MAX_ATTEMPTS,
  VERIFICATION_RESEND_COOLDOWN_MS,
  generateVerificationCode,
  generateVerificationToken,
  hashVerificationCode,
  verificationCodeMatches,
} from './verification-code';

const PASSWORD_SALT_ROUNDS = 10;

// Email registrations get a typed 6-digit code. Phone registrations get
// verified by opening a Telegram deep link — the bot process (which shares
// this same database) flips isVerified once the link is opened, since only
// it can actually deliver a message to a Telegram chat.
export type VerificationMethod = 'email' | 'telegram';

export interface PendingVerification {
  identifier: string;
  method: VerificationMethod;
  // Only set for the 'telegram' method — the frontend embeds this in the
  // bot deep link (https://t.me/<bot>?start=verify_<token>).
  token?: string;
}

const CODE_FIELDS = [
  'id',
  'telegramId',
  'email',
  'username',
  'firstName',
  'lastName',
  'phone',
  'avatarUrl',
  'isVerified',
  'partnerStatus',
  'partnerRequestedAt',
  'verificationCodeHash',
  'verificationCodeExpiresAt',
  'verificationCodeSentAt',
  'verificationAttempts',
] as const;

@Injectable()
export class CredentialsAuthService {
  constructor(
    private readonly jwtService: JwtService,
    private readonly emailService: EmailService,
    @InjectRepository(User)
    private readonly users: Repository<User>,
  ) {}

  async register(dto: RegisterDto): Promise<PendingVerification> {
    if (dto.email) {
      const existing = await this.users.findOne({ where: { email: dto.email } });
      if (existing) {
        throw new ConflictException('Email is already registered');
      }
    }
    if (dto.phone) {
      const existing = await this.users.findOne({ where: { phone: dto.phone } });
      if (existing) {
        throw new ConflictException('Phone is already registered');
      }
    }

    const passwordHash = await bcrypt.hash(dto.password, PASSWORD_SALT_ROUNDS);
    const created = this.users.create({
      telegramId: null,
      email: dto.email ?? null,
      phone: dto.phone ?? null,
      passwordHash,
      firstName: dto.firstName ?? null,
      username: null,
      lastName: null,
      avatarUrl: null,
      isVerified: false,
    });
    const saved = await this.users.save(created);

    try {
      return await this.issueVerification(saved);
    } catch (error) {
      // Don't leave an unverifiable account holding the email/phone hostage.
      await this.users.remove(saved);
      throw error;
    }
  }

  async login(dto: LoginDto): Promise<{ token: string; user: PublicUser } | PendingVerification> {
    const user = await this.users.findOne({
      where: [{ email: dto.identifier }, { phone: dto.identifier }],
      select: [
        'id',
        'telegramId',
        'email',
        'username',
        'firstName',
        'lastName',
        'phone',
        'avatarUrl',
        'isVerified',
        'partnerStatus',
        'passwordHash',
      ],
    });

    if (!user?.passwordHash) {
      throw new UnauthorizedException('Invalid credentials');
    }

    const valid = await bcrypt.compare(dto.password, user.passwordHash);
    if (!valid) {
      throw new UnauthorizedException('Invalid credentials');
    }

    if (!user.isVerified) {
      return { identifier: dto.identifier, method: user.email ? 'email' : 'telegram' };
    }

    return this.issueSession(user);
  }

  async verifyCode(dto: VerifyCodeDto): Promise<{ token: string; user: PublicUser }> {
    const user = await this.findByIdentifierWithCode(dto.identifier);
    if (!user) {
      throw new NotFoundException('Account not found');
    }
    if (user.isVerified) {
      throw new ConflictException('Account is already verified');
    }
    if (!user.verificationCodeHash || !user.verificationCodeExpiresAt) {
      throw new BadRequestException('No verification code pending. Please request a new one.');
    }
    if (user.verificationCodeExpiresAt.getTime() < Date.now()) {
      throw new BadRequestException('Code has expired. Please request a new one.');
    }
    if (user.verificationAttempts >= VERIFICATION_MAX_ATTEMPTS) {
      throw new BadRequestException('Too many attempts. Please request a new code.');
    }

    if (!verificationCodeMatches(dto.code, user.verificationCodeHash)) {
      user.verificationAttempts += 1;
      await this.users.save(user);
      throw new UnauthorizedException('Invalid code');
    }

    user.isVerified = true;
    user.verificationCodeHash = null;
    user.verificationCodeExpiresAt = null;
    user.verificationCodeSentAt = null;
    user.verificationAttempts = 0;
    await this.users.save(user);

    return this.issueSession(user);
  }

  async resendCode(dto: ResendCodeDto): Promise<PendingVerification> {
    const user = await this.findByIdentifierWithCode(dto.identifier);
    if (!user) {
      throw new NotFoundException('Account not found');
    }
    if (user.isVerified) {
      throw new ConflictException('Account is already verified');
    }
    if (
      user.verificationCodeSentAt &&
      Date.now() - user.verificationCodeSentAt.getTime() < VERIFICATION_RESEND_COOLDOWN_MS
    ) {
      throw new HttpException(
        'Please wait before requesting another code',
        HttpStatus.TOO_MANY_REQUESTS,
      );
    }

    return this.issueVerification(user);
  }

  // Polled by the frontend while waiting for the Telegram bot to confirm a
  // pending phone verification.
  async getVerificationStatus(dto: VerificationStatusDto): Promise<{ verified: boolean }> {
    const user = await this.users.findOne({
      where: [{ email: dto.identifier }, { phone: dto.identifier }],
      select: ['id', 'isVerified'],
    });
    if (!user) {
      throw new NotFoundException('Account not found');
    }
    return { verified: user.isVerified };
  }

  private async issueVerification(user: User): Promise<PendingVerification> {
    if (user.email) {
      const code = generateVerificationCode();
      user.verificationCodeHash = hashVerificationCode(code);
      user.verificationCodeExpiresAt = new Date(Date.now() + VERIFICATION_CODE_TTL_MS);
      user.verificationCodeSentAt = new Date();
      user.verificationAttempts = 0;
      await this.users.save(user);

      await this.emailService.sendVerificationCode(user.email, code);
      return { identifier: user.email, method: 'email' };
    }

    // Phone: nothing to send yet — the token only becomes useful once the
    // user opens it in Telegram, at which point the bot process (sharing
    // this database) verifies it directly.
    const token = generateVerificationToken();
    user.verificationCodeHash = hashVerificationCode(token);
    user.verificationCodeExpiresAt = new Date(Date.now() + VERIFICATION_CODE_TTL_MS);
    user.verificationCodeSentAt = new Date();
    user.verificationAttempts = 0;
    await this.users.save(user);

    return { identifier: user.phone as string, method: 'telegram', token };
  }

  private findByIdentifierWithCode(identifier: string): Promise<User | null> {
    return this.users.findOne({
      where: [{ email: identifier }, { phone: identifier }],
      select: [...CODE_FIELDS],
    });
  }

  private issueSession(user: User): { token: string; user: PublicUser } {
    const token = this.jwtService.sign({ sub: user.id, role: 'user' });
    return { token, user: toPublicUser(user) };
  }
}
