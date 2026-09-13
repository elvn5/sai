import { User } from '@app/database';
import {
  type CanActivate,
  type ExecutionContext,
  Injectable,
  UnauthorizedException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { InjectRepository } from '@nestjs/typeorm';
import type { Request } from 'express';
import type { Repository } from 'typeorm';

interface UserJwtPayload {
  sub: string;
  role: string;
}

@Injectable()
export class UserJwtAuthGuard implements CanActivate {
  constructor(
    private readonly jwtService: JwtService,
    @InjectRepository(User)
    private readonly users: Repository<User>,
  ) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const request = context.switchToHttp().getRequest<Request>();
    const token = request.cookies?.user_token as string | undefined;

    if (!token) {
      throw new UnauthorizedException('Not authenticated');
    }

    let payload: UserJwtPayload;
    try {
      payload = this.jwtService.verify<UserJwtPayload>(token);
    } catch {
      throw new UnauthorizedException('Invalid or expired session');
    }

    if (payload.role !== 'user') {
      throw new UnauthorizedException('Invalid session');
    }

    const user = await this.users.findOne({ where: { id: payload.sub } });
    if (!user) {
      throw new UnauthorizedException('User not found');
    }

    (request as Request & { currentUser: User }).currentUser = user;
    return true;
  }
}
