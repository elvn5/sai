import { UserAuthModule } from '@/user-auth/user-auth.module';
import { Guide } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { GuidesController } from './guides.controller';
import { GuidesService } from './guides.service';

@Module({
  imports: [TypeOrmModule.forFeature([Guide]), UserAuthModule],
  controllers: [GuidesController],
  providers: [GuidesService],
})
export class GuidesModule {}
