import { AuthModule } from '@/auth/auth.module';
import { SupportInfo } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { SupportController } from './support.controller';
import { SupportService } from './support.service';

@Module({
  imports: [TypeOrmModule.forFeature([SupportInfo]), AuthModule],
  controllers: [SupportController],
  providers: [SupportService],
})
export class SupportModule {}
