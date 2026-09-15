import { JwtAuthGuard } from '@/auth/jwt-auth.guard';
import { CurrentUser } from '@/user-auth/current-user.decorator';
import { type PublicUser, toPublicUser } from '@/user-auth/public-user';
import { UserJwtAuthGuard } from '@/user-auth/user-jwt-auth.guard';
import { User } from '@app/database';
import { Controller, Get, Param, Post, UseGuards } from '@nestjs/common';
import { PartnersService } from './partners.service';

@Controller('partners')
export class PartnersController {
  constructor(private readonly service: PartnersService) {}

  @Post('apply')
  @UseGuards(UserJwtAuthGuard)
  async apply(@CurrentUser() user: User): Promise<{ user: PublicUser }> {
    const updated = await this.service.apply(user.id);
    return { user: toPublicUser(updated) };
  }

  @Get('requests')
  @UseGuards(JwtAuthGuard)
  findPending() {
    return this.service.findPending();
  }

  @Post(':userId/approve')
  @UseGuards(JwtAuthGuard)
  approve(@Param('userId') userId: string) {
    return this.service.approve(userId);
  }

  @Post(':userId/reject')
  @UseGuards(JwtAuthGuard)
  reject(@Param('userId') userId: string) {
    return this.service.reject(userId);
  }
}
