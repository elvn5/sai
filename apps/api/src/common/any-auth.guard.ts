import { JwtAuthGuard } from '@/auth/jwt-auth.guard';
import { UserJwtAuthGuard } from '@/user-auth/user-jwt-auth.guard';
import { type CanActivate, type ExecutionContext, Injectable } from '@nestjs/common';

/**
 * Accepts either an admin session or a regular user session — for endpoints
 * (like image uploads) both the admin panel and the client site call.
 */
@Injectable()
export class AnyAuthGuard implements CanActivate {
  constructor(
    private readonly adminGuard: JwtAuthGuard,
    private readonly userGuard: UserJwtAuthGuard,
  ) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    try {
      return await this.adminGuard.canActivate(context);
    } catch {
      return this.userGuard.canActivate(context);
    }
  }
}
