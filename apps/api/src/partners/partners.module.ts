import { AuthModule } from '@/auth/auth.module';
import { UserAuthModule } from '@/user-auth/user-auth.module';
import { User } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { PartnersController } from './partners.controller';
import { PartnersService } from './partners.service';

@Module({
  imports: [TypeOrmModule.forFeature([User]), AuthModule, UserAuthModule],
  controllers: [PartnersController],
  providers: [PartnersService],
})
export class PartnersModule {}
