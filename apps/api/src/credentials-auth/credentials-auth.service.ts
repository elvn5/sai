import { type PublicUser, toPublicUser } from '@/user-auth/public-user';
import { User } from '@app/database';
import { ConflictException, Injectable, UnauthorizedException } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { InjectRepository } from '@nestjs/typeorm';
import * as bcrypt from 'bcryptjs';
import type { Repository } from 'typeorm';
import type { LoginDto, RegisterDto } from './credentials-auth.dto';

const PASSWORD_SALT_ROUNDS = 10;

@Injectable()
export class CredentialsAuthService {
  constructor(
    private readonly jwtService: JwtService,
    @InjectRepository(User)
    private readonly users: Repository<User>,
  ) {}

  async register(dto: RegisterDto): Promise<{ token: string; user: PublicUser }> {
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
    });
    const saved = await this.users.save(created);
    return this.issueSession(saved);
  }

  async login(dto: LoginDto): Promise<{ token: string; user: PublicUser }> {
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

    return this.issueSession(user);
  }

  private issueSession(user: User): { token: string; user: PublicUser } {
    const token = this.jwtService.sign({ sub: user.id, role: 'user' });
    return { token, user: toPublicUser(user) };
  }
}
