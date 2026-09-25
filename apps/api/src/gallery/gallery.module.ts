import { StorageModule } from '@/storage/storage.module';
import { UserAuthModule } from '@/user-auth/user-auth.module';
import { GalleryItem } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { GalleryController } from './gallery.controller';
import { GalleryService } from './gallery.service';

@Module({
  imports: [TypeOrmModule.forFeature([GalleryItem]), UserAuthModule, StorageModule],
  controllers: [GalleryController],
  providers: [GalleryService],
})
export class GalleryModule {}
