import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import type { PublicUser } from '@/user-auth/public-user';
import { setUserSessionCookie } from '@/user-auth/session-cookie';
import { Body, Controller, HttpCode, Post, Res } from '@nestjs/common';
import type { Response } from 'express';
import {
  type TelegramMiniAppLoginDto,
  type TelegramWidgetLoginDto,
  telegramMiniAppLoginSchema,
  telegramWidgetLoginSchema,
} from './telegram-auth.dto';
import { TelegramAuthService } from './telegram-auth.service';

@Controller('auth/telegram')
export class TelegramAuthController {
  constructor(private readonly service: TelegramAuthService) {}

  @Post('widget')
  @HttpCode(200)
  async loginWithWidget(
    @Body(new ZodValidationPipe(telegramWidgetLoginSchema)) dto: TelegramWidgetLoginDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<{ user: PublicUser }> {
    const { token, user } = await this.service.loginWithWidget(dto);
    setUserSessionCookie(res, token);
    return { user };
  }

  @Post('miniapp')
  @HttpCode(200)
  async loginWithMiniApp(
    @Body(new ZodValidationPipe(telegramMiniAppLoginSchema)) dto: TelegramMiniAppLoginDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<{ user: PublicUser }> {
    const { token, user } = await this.service.loginWithMiniApp(dto.initData);
    setUserSessionCookie(res, token);
    return { user };
  }
}
