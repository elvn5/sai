import { ListingTargetsModule } from '@/listing-targets/listing-targets.module';
import { UserAuthModule } from '@/user-auth/user-auth.module';
import { LocalGuide } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { LocalGuidesController } from './local-guides.controller';
import { LocalGuidesService } from './local-guides.service';

@Module({
  imports: [TypeOrmModule.forFeature([LocalGuide]), UserAuthModule, ListingTargetsModule],
  controllers: [LocalGuidesController],
  providers: [LocalGuidesService],
})
export class LocalGuidesModule {}
