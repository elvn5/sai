import { PartnerStatus, User } from '@app/database';
import {
  type CanActivate,
  type ExecutionContext,
  ForbiddenException,
  Injectable,
} from '@nestjs/common';
import type { Request } from 'express';

// Runs after UserJwtAuthGuard, which attaches currentUser to the request.
@Injectable()
export class PartnerGuard implements CanActivate {
  canActivate(context: ExecutionContext): boolean {
    const request = context.switchToHttp().getRequest<Request & { currentUser: User }>();
    if (request.currentUser?.partnerStatus !== PartnerStatus.APPROVED) {
      throw new ForbiddenException('Only approved partners can do this');
    }
    return true;
  }
}
