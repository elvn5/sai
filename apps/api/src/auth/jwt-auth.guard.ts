import {
  type CanActivate,
  type ExecutionContext,
  Injectable,
  UnauthorizedException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import type { Request } from 'express';

@Injectable()
export class JwtAuthGuard implements CanActivate {
  constructor(private readonly jwtService: JwtService) {}

  canActivate(context: ExecutionContext): boolean {
    const request = context.switchToHttp().getRequest<Request>();
    const token = request.cookies?.admin_token as string | undefined;

    if (!token) {
      throw new UnauthorizedException('Not authenticated');
    }

    try {
      this.jwtService.verify(token);
      return true;
    } catch {
      throw new UnauthorizedException('Invalid or expired session');
    }
  }
}
