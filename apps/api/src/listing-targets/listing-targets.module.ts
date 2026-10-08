import { Module } from '@nestjs/common';
import { ListingTargetsService } from './listing-targets.service';

@Module({
  providers: [ListingTargetsService],
  exports: [ListingTargetsService],
})
export class ListingTargetsModule {}
