import { User } from '@app/database';
import { Controller, Get, HttpCode, Post, Req, Res, UseGuards } from '@nestjs/common';
import type { Request, Response } from 'express';
import { type PublicUser, toPublicUser } from './public-user';
import { UserJwtAuthGuard } from './user-jwt-auth.guard';

@Controller('auth/user')
export class UserAuthController {
  @Get('me')
  @UseGuards(UserJwtAuthGuard)
  me(@Req() req: Request): { user: PublicUser } {
    const currentUser = (req as Request & { currentUser: User }).currentUser;
    return { user: toPublicUser(currentUser) };
  }

  @Post('logout')
  @HttpCode(200)
  logout(@Res({ passthrough: true }) res: Response): { success: true } {
    res.clearCookie('user_token');
    return { success: true };
  }
}
