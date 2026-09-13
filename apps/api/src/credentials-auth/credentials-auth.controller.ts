import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import type { PublicUser } from '@/user-auth/public-user';
import { setUserSessionCookie } from '@/user-auth/session-cookie';
import { Body, Controller, HttpCode, Post, Res } from '@nestjs/common';
import type { Response } from 'express';
import {
  type LoginDto,
  type RegisterDto,
  loginSchema,
  registerSchema,
} from './credentials-auth.dto';
import { CredentialsAuthService } from './credentials-auth.service';

@Controller('auth/user')
export class CredentialsAuthController {
  constructor(private readonly service: CredentialsAuthService) {}

  @Post('register')
  @HttpCode(200)
  async register(
    @Body(new ZodValidationPipe(registerSchema)) dto: RegisterDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<{ user: PublicUser }> {
    const { token, user } = await this.service.register(dto);
    setUserSessionCookie(res, token);
    return { user };
  }

  @Post('login')
  @HttpCode(200)
  async login(
    @Body(new ZodValidationPipe(loginSchema)) dto: LoginDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<{ user: PublicUser }> {
    const { token, user } = await this.service.login(dto);
    setUserSessionCookie(res, token);
    return { user };
  }
}
