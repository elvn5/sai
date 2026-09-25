import { AuthModule } from '@/auth/auth.module';
import { AnyAuthGuard } from '@/common/any-auth.guard';
import { UserAuthModule } from '@/user-auth/user-auth.module';
import { Module } from '@nestjs/common';
import { StorageController } from './storage.controller';
import { StorageService } from './storage.service';

@Module({
  imports: [AuthModule, UserAuthModule],
  controllers: [StorageController],
  providers: [StorageService, AnyAuthGuard],
  exports: [StorageService],
})
export class StorageModule {}
