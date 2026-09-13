import { AuthModule } from '@/auth/auth.module';
import { User } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { UserAuthController } from './user-auth.controller';
import { UserJwtAuthGuard } from './user-jwt-auth.guard';

@Module({
  imports: [TypeOrmModule.forFeature([User]), AuthModule],
  controllers: [UserAuthController],
  providers: [UserJwtAuthGuard],
  exports: [UserJwtAuthGuard],
})
export class UserAuthModule {}
