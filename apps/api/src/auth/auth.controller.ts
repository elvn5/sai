import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { Body, Controller, Get, HttpCode, Post, Res, UseGuards } from '@nestjs/common';
import type { Response } from 'express';
import { type LoginDto, loginSchema } from './auth.dto';
import { AuthService } from './auth.service';
import { JwtAuthGuard } from './jwt-auth.guard';

const isProduction = process.env.NODE_ENV === 'production';

@Controller('auth')
export class AuthController {
  constructor(private readonly authService: AuthService) {}

  @Post('login')
  @HttpCode(200)
  async login(
    @Body(new ZodValidationPipe(loginSchema)) dto: LoginDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<{ success: true }> {
    const token = await this.authService.login(dto.password);

    res.cookie('admin_token', token, {
      httpOnly: true,
      secure: isProduction,
      sameSite: 'lax',
      maxAge: 12 * 60 * 60 * 1000,
    });

    return { success: true };
  }

  @Post('logout')
  @HttpCode(200)
  logout(@Res({ passthrough: true }) res: Response): { success: true } {
    res.clearCookie('admin_token');
    return { success: true };
  }

  @Get('me')
  @UseGuards(JwtAuthGuard)
  me(): { authenticated: true } {
    return { authenticated: true };
  }
}
