import { PartnerGuard } from '@/user-auth/partner.guard';
import { UserAuthModule } from '@/user-auth/user-auth.module';
import { Tour } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { ToursController } from './tours.controller';
import { ToursService } from './tours.service';

@Module({
  imports: [TypeOrmModule.forFeature([Tour]), UserAuthModule],
  controllers: [ToursController],
  providers: [ToursService, PartnerGuard],
})
export class ToursModule {}
