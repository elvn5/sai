import { ListingTargetsModule } from '@/listing-targets/listing-targets.module';
import { UserAuthModule } from '@/user-auth/user-auth.module';
import { Hotel } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { HotelsController } from './hotels.controller';
import { HotelsService } from './hotels.service';

@Module({
  imports: [TypeOrmModule.forFeature([Hotel]), UserAuthModule, ListingTargetsModule],
  controllers: [HotelsController],
  providers: [HotelsService],
})
export class HotelsModule {}
