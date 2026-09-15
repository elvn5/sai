import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import type { PublicUser } from '@/user-auth/public-user';
import { setUserSessionCookie } from '@/user-auth/session-cookie';
import { Body, Controller, HttpCode, Post, Res } from '@nestjs/common';
import type { Response } from 'express';
import {
  type LoginDto,
  type RegisterDto,
  type ResendCodeDto,
  type VerifyCodeDto,
  loginSchema,
  registerSchema,
  resendCodeSchema,
  verifyCodeSchema,
} from './credentials-auth.dto';
import type { PendingVerification } from './credentials-auth.service';
import { CredentialsAuthService } from './credentials-auth.service';

@Controller('auth/user')
export class CredentialsAuthController {
  constructor(private readonly service: CredentialsAuthService) {}

  @Post('register')
  @HttpCode(200)
  async register(
    @Body(new ZodValidationPipe(registerSchema)) dto: RegisterDto,
  ): Promise<{ verification: PendingVerification }> {
    const verification = await this.service.register(dto);
    return { verification };
  }

  @Post('login')
  @HttpCode(200)
  async login(
    @Body(new ZodValidationPipe(loginSchema)) dto: LoginDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<{ user: PublicUser } | { verification: PendingVerification }> {
    const result = await this.service.login(dto);
    if ('token' in result) {
      setUserSessionCookie(res, result.token);
      return { user: result.user };
    }

    return { verification: result };
  }

  @Post('verify')
  @HttpCode(200)
  async verify(
    @Body(new ZodValidationPipe(verifyCodeSchema)) dto: VerifyCodeDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<{ user: PublicUser }> {
    const { token, user } = await this.service.verifyCode(dto);
    setUserSessionCookie(res, token);
    return { user };
  }

  @Post('resend-code')
  @HttpCode(200)
  async resendCode(
    @Body(new ZodValidationPipe(resendCodeSchema)) dto: ResendCodeDto,
  ): Promise<{ verification: PendingVerification }> {
    const verification = await this.service.resendCode(dto);
    return { verification };
  }
}
