import { AuthModule } from '@/auth/auth.module';
import { Parcel } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { ParcelsController } from './parcels.controller';
import { ParcelsService } from './parcels.service';

@Module({
  imports: [TypeOrmModule.forFeature([Parcel]), AuthModule],
  controllers: [ParcelsController],
  providers: [ParcelsService],
})
export class ParcelsModule {}
