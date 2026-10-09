import { UsersGrpcModule } from '@/users-grpc/users-grpc.module';
import { Module } from '@nestjs/common';
import { SupportService } from './support.service';

@Module({
  imports: [UsersGrpcModule],
  providers: [SupportService],
  exports: [SupportService],
})
export class SupportModule {}
