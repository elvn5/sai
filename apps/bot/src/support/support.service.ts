import { type SupportInfo, UsersGrpcClient } from '@/users-grpc/users-grpc.client';
import { Injectable } from '@nestjs/common';

export interface SupportContact {
  message: string;
  phone: string | null;
  telegramContact: string | null;
  email: string | null;
  workingHours: string | null;
}

// Support contacts are edited in the admin panel and served by the users
// service (which falls back to a default message).
@Injectable()
export class SupportService {
  constructor(private readonly users: UsersGrpcClient) {}

  async get(): Promise<SupportContact> {
    const info: SupportInfo = await this.users.getSupportInfo();
    return {
      message: info.message,
      phone: info.phone ?? null,
      telegramContact: info.telegramContact ?? null,
      email: info.email ?? null,
      workingHours: info.workingHours ?? null,
    };
  }
}
