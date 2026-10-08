import {
  type CanActivate,
  type ExecutionContext,
  Injectable,
  UnauthorizedException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import type { Request } from 'express';

interface AdminJwtPayload {
  role?: string;
}

@Injectable()
export class JwtAuthGuard implements CanActivate {
  constructor(private readonly jwtService: JwtService) {}

  canActivate(context: ExecutionContext): boolean {
    const request = context.switchToHttp().getRequest<Request>();
    const token = request.cookies?.admin_token as string | undefined;

    if (!token) {
      throw new UnauthorizedException('Not authenticated');
    }

    let payload: AdminJwtPayload;
    try {
      payload = this.jwtService.verify<AdminJwtPayload>(token);
    } catch {
      throw new UnauthorizedException('Invalid or expired session');
    }

    // Site-user sessions are signed with the same secret, so a valid
    // signature alone doesn't make a token an admin session.
    if (payload.role !== 'admin') {
      throw new UnauthorizedException('Invalid session');
    }
    return true;
  }
}
