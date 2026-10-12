import { resolve } from 'node:path';
import { promisify } from 'node:util';
import { type ServiceError, credentials, loadPackageDefinition, status } from '@grpc/grpc-js';
import { loadSync } from '@grpc/proto-loader';
import { Injectable, type OnModuleDestroy } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

// The users service's internal API (services/proto/users/v1/users.proto),
// loaded at runtime. The bot no longer touches a database: accounts and
// support contacts belong to the users service.
const DEFAULT_PROTO_PATH = resolve(__dirname, '../../../../services/proto/users/v1/users.proto');

export interface TelegramAccount {
  id: string;
  firstName?: string;
  username?: string;
  phone?: string;
}

export interface SupportInfo {
  message: string;
  phone?: string;
  telegramContact?: string;
  email?: string;
  workingHours?: string;
}

interface UpsertTelegramUserRequest {
  telegramId: string;
  username?: string;
  firstName?: string;
  lastName?: string;
}

interface LinkTelegramRequest {
  token: string;
  telegramId: string;
  username?: string;
  moveFromOtherAccount?: boolean;
}

// How a link attempt ended.
export type LinkTelegramResult =
  | { status: 'linked'; account: TelegramAccount }
  // Unknown, expired or already-used token.
  | { status: 'invalid' }
  // The Telegram account belongs to another account with its own sign-in.
  | { status: 'taken' }
  // It belongs to a Telegram-only account; linking moves it from there.
  | { status: 'needsMove' };

interface UpdateProfileRequest {
  userId: string;
  firstName?: string;
  phone?: string;
}

type Unary<Req, Res> = (
  request: Req,
  callback: (error: ServiceError | null, response: Res) => void,
) => void;

interface RawClient {
  upsertTelegramUser: Unary<UpsertTelegramUserRequest, { account: TelegramAccount }>;
  confirmTelegramVerification: Unary<{ token: string }, { account: TelegramAccount }>;
  linkTelegram: Unary<LinkTelegramRequest, { account: TelegramAccount }>;
  updateProfile: Unary<UpdateProfileRequest, { account: TelegramAccount }>;
  getSupportInfo: Unary<Record<string, never>, { support: SupportInfo }>;
  close(): void;
}

export class PhoneTakenError extends Error {}

@Injectable()
export class UsersGrpcClient implements OnModuleDestroy {
  private readonly client: RawClient;

  constructor(configService: ConfigService) {
    const protoPath = configService.get<string>('USERS_PROTO_PATH') ?? DEFAULT_PROTO_PATH;
    const definition = loadSync(protoPath, {
      keepCase: false,
      longs: String,
      defaults: false,
      oneofs: true,
    });
    const pkg = loadPackageDefinition(definition) as unknown as {
      users: {
        v1: {
          UsersService: new (
            addr: string,
            creds: ReturnType<typeof credentials.createInsecure>,
          ) => RawClient;
        };
      };
    };
    const address = configService.getOrThrow<string>('USERS_GRPC_ADDR');
    this.client = new pkg.users.v1.UsersService(address, credentials.createInsecure());
  }

  async upsertTelegramUser(request: UpsertTelegramUserRequest): Promise<TelegramAccount> {
    const res = await this.call(this.client.upsertTelegramUser, request);
    return res.account;
  }

  // Null for an unknown, expired or already-used token.
  async confirmTelegramVerification(token: string): Promise<TelegramAccount | null> {
    try {
      const res = await this.call(this.client.confirmTelegramVerification, { token });
      return res.account;
    } catch (error) {
      if ((error as ServiceError).code === status.NOT_FOUND) return null;
      throw error;
    }
  }

  async linkTelegram(request: LinkTelegramRequest): Promise<LinkTelegramResult> {
    try {
      const res = await this.call(this.client.linkTelegram, request);
      return { status: 'linked', account: res.account };
    } catch (error) {
      switch ((error as ServiceError).code) {
        case status.NOT_FOUND:
          return { status: 'invalid' };
        case status.ALREADY_EXISTS:
          return { status: 'taken' };
        case status.FAILED_PRECONDITION:
          return { status: 'needsMove' };
        default:
          throw error;
      }
    }
  }

  async updateProfile(request: UpdateProfileRequest): Promise<TelegramAccount> {
    try {
      const res = await this.call(this.client.updateProfile, request);
      return res.account;
    } catch (error) {
      if ((error as ServiceError).code === status.ALREADY_EXISTS) throw new PhoneTakenError();
      throw error;
    }
  }

  async getSupportInfo(): Promise<SupportInfo> {
    const res = await this.call(this.client.getSupportInfo, {});
    return res.support;
  }

  onModuleDestroy(): void {
    this.client.close();
  }

  private call<Req, Res>(method: Unary<Req, Res>, request: Req): Promise<Res> {
    return promisify(method).call(this.client, request) as Promise<Res>;
  }
}
