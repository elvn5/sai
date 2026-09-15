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
  // Re-export AuthModule (JwtModule) too, since UserJwtAuthGuard depends on
  // JwtService — modules that only import UserAuthModule to use the guard
  // need that dependency resolvable as well.
  exports: [UserJwtAuthGuard, TypeOrmModule, AuthModule],
})
export class UserAuthModule {}
