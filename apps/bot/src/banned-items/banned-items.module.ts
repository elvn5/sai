import { BannedItem } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { BannedItemsService } from './banned-items.service';

@Module({
  imports: [TypeOrmModule.forFeature([BannedItem])],
  providers: [BannedItemsService],
  exports: [BannedItemsService],
})
export class BannedItemsModule {}
