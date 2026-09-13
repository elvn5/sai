import { AuthModule } from '@/auth/auth.module';
import { Guide } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { GuidesController } from './guides.controller';
import { GuidesService } from './guides.service';

@Module({
  imports: [TypeOrmModule.forFeature([Guide]), AuthModule],
  controllers: [GuidesController],
  providers: [GuidesService],
})
export class GuidesModule {}
