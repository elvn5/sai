import { Parcel } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { ParcelsService } from './parcels.service';

@Module({
  imports: [TypeOrmModule.forFeature([Parcel])],
  providers: [ParcelsService],
  exports: [ParcelsService],
})
export class ParcelsModule {}
