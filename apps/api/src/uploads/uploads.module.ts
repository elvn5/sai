import { AuthModule } from '@/auth/auth.module';
import { AnyAuthGuard } from '@/common/any-auth.guard';
import { UserAuthModule } from '@/user-auth/user-auth.module';
import { Module } from '@nestjs/common';
import { UploadsController } from './uploads.controller';

@Module({
  imports: [AuthModule, UserAuthModule],
  controllers: [UploadsController],
  providers: [AnyAuthGuard],
})
export class UploadsModule {}
