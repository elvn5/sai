import { AuthModule } from '@/auth/auth.module';
import { BannedItem } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { BannedItemsController } from './banned-items.controller';
import { BannedItemsService } from './banned-items.service';

@Module({
  imports: [TypeOrmModule.forFeature([BannedItem]), AuthModule],
  controllers: [BannedItemsController],
  providers: [BannedItemsService],
})
export class BannedItemsModule {}
