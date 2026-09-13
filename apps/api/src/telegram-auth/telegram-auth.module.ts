import { AuthModule } from '@/auth/auth.module';
import { User } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { TelegramAuthController } from './telegram-auth.controller';
import { TelegramAuthService } from './telegram-auth.service';
import { UserJwtAuthGuard } from './user-jwt-auth.guard';

@Module({
  imports: [TypeOrmModule.forFeature([User]), AuthModule],
  controllers: [TelegramAuthController],
  providers: [TelegramAuthService, UserJwtAuthGuard],
  exports: [UserJwtAuthGuard],
})
export class TelegramAuthModule {}
