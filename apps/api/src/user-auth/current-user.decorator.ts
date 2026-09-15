import { User } from '@app/database';
import { type ExecutionContext, createParamDecorator } from '@nestjs/common';
import type { Request } from 'express';

// Reads the user attached by UserJwtAuthGuard — only valid on routes guarded
// by it.
export const CurrentUser = createParamDecorator((_data: unknown, ctx: ExecutionContext): User => {
  const request = ctx.switchToHttp().getRequest<Request & { currentUser: User }>();
  return request.currentUser;
});
