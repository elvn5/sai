import { Module } from '@nestjs/common';
import { UsersGrpcClient } from './users-grpc.client';

@Module({
  providers: [UsersGrpcClient],
  exports: [UsersGrpcClient],
})
export class UsersGrpcModule {}
