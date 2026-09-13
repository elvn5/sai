import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { User } from '@app/database';
import { Body, Controller, Get, HttpCode, Post, Req, Res, UseGuards } from '@nestjs/common';
import type { Request, Response } from 'express';
import {
  type PublicTelegramUser,
  type TelegramMiniAppLoginDto,
  type TelegramWidgetLoginDto,
  telegramMiniAppLoginSchema,
  telegramWidgetLoginSchema,
} from './telegram-auth.dto';
import { TelegramAuthService } from './telegram-auth.service';
import { UserJwtAuthGuard } from './user-jwt-auth.guard';

const isProduction = process.env.NODE_ENV === 'production';
const USER_TOKEN_MAX_AGE_MS = 30 * 24 * 60 * 60 * 1000;

@Controller('auth/telegram')
export class TelegramAuthController {
  constructor(private readonly service: TelegramAuthService) {}

  @Post('widget')
  @HttpCode(200)
  async loginWithWidget(
    @Body(new ZodValidationPipe(telegramWidgetLoginSchema)) dto: TelegramWidgetLoginDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<{ user: PublicTelegramUser }> {
    const { token, user } = await this.service.loginWithWidget(dto);
    this.setCookie(res, token);
    return { user };
  }

  @Post('miniapp')
  @HttpCode(200)
  async loginWithMiniApp(
    @Body(new ZodValidationPipe(telegramMiniAppLoginSchema)) dto: TelegramMiniAppLoginDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<{ user: PublicTelegramUser }> {
    const { token, user } = await this.service.loginWithMiniApp(dto.initData);
    this.setCookie(res, token);
    return { user };
  }

  @Get('me')
  @UseGuards(UserJwtAuthGuard)
  me(@Req() req: Request): { user: PublicTelegramUser } {
    const currentUser = (req as Request & { currentUser: User }).currentUser;
    return {
      user: {
        id: currentUser.id,
        telegramId: currentUser.telegramId,
        username: currentUser.username,
        firstName: currentUser.firstName,
        lastName: currentUser.lastName,
        avatarUrl: currentUser.avatarUrl,
      },
    };
  }

  @Post('logout')
  @HttpCode(200)
  logout(@Res({ passthrough: true }) res: Response): { success: true } {
    res.clearCookie('user_token');
    return { success: true };
  }

  private setCookie(res: Response, token: string): void {
    res.cookie('user_token', token, {
      httpOnly: true,
      secure: isProduction,
      sameSite: 'lax',
      maxAge: USER_TOKEN_MAX_AGE_MS,
    });
  }
}
