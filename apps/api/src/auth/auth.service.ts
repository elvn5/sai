import { timingSafeEqual } from 'node:crypto';
import { Injectable, UnauthorizedException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { JwtService } from '@nestjs/jwt';

@Injectable()
export class AuthService {
  constructor(
    private readonly jwtService: JwtService,
    private readonly configService: ConfigService,
  ) {}

  async login(password: string): Promise<string> {
    const adminPassword = this.configService.get<string>('ADMIN_PASSWORD', { infer: true });
    if (!adminPassword) {
      throw new UnauthorizedException('Admin password is not configured');
    }

    if (!this.isEqual(password, adminPassword)) {
      throw new UnauthorizedException('Invalid password');
    }

    return this.jwtService.sign({ role: 'admin' });
  }

  private isEqual(a: string, b: string): boolean {
    const bufA = Buffer.from(a);
    const bufB = Buffer.from(b);
    if (bufA.length !== bufB.length) {
      return false;
    }
    return timingSafeEqual(bufA, bufB);
  }
}
