import { SupportInfo } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { SupportService } from './support.service';

@Module({
  imports: [TypeOrmModule.forFeature([SupportInfo])],
  providers: [SupportService],
  exports: [SupportService],
})
export class SupportModule {}
