import { AuthModule } from '@/auth/auth.module';
import { Module } from '@nestjs/common';
import { UploadsController } from './uploads.controller';

@Module({
  imports: [AuthModule],
  controllers: [UploadsController],
})
export class UploadsModule {}
