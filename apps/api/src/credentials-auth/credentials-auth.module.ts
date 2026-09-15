import { AuthModule } from '@/auth/auth.module';
import { NotificationsModule } from '@/notifications/notifications.module';
import { User } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { CredentialsAuthController } from './credentials-auth.controller';
import { CredentialsAuthService } from './credentials-auth.service';

@Module({
  imports: [TypeOrmModule.forFeature([User]), AuthModule, NotificationsModule],
  controllers: [CredentialsAuthController],
  providers: [CredentialsAuthService],
})
export class CredentialsAuthModule {}
