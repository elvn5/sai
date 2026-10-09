import { UsersGrpcModule } from '@/users-grpc/users-grpc.module';
import { Module } from '@nestjs/common';
import { UsersService } from './users.service';

@Module({
  imports: [UsersGrpcModule],
  providers: [UsersService],
  exports: [UsersService],
})
export class UsersModule {}
