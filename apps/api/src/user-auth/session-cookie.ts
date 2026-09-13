import type { Response } from 'express';

const isProduction = process.env.NODE_ENV === 'production';
const USER_TOKEN_MAX_AGE_MS = 30 * 24 * 60 * 60 * 1000;

export function setUserSessionCookie(res: Response, token: string): void {
  res.cookie('user_token', token, {
    httpOnly: true,
    secure: isProduction,
    sameSite: 'lax',
    maxAge: USER_TOKEN_MAX_AGE_MS,
  });
}
