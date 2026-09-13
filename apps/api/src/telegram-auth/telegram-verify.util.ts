import { createHash, createHmac } from 'node:crypto';

export function verifyTelegramWidgetHash(
  payload: Record<string, string | number | undefined>,
  botToken: string,
): boolean {
  const { hash, ...rest } = payload;
  if (typeof hash !== 'string') return false;

  const dataCheckString = Object.keys(rest)
    .filter((key) => rest[key] !== undefined)
    .sort()
    .map((key) => `${key}=${rest[key]}`)
    .join('\n');

  const secretKey = createHash('sha256').update(botToken).digest();
  const computed = createHmac('sha256', secretKey).update(dataCheckString).digest('hex');
  return computed === hash;
}

export function verifyTelegramInitData(initData: string, botToken: string): URLSearchParams | null {
  const params = new URLSearchParams(initData);
  const hash = params.get('hash');
  if (!hash) return null;

  params.delete('hash');
  const dataCheckString = [...params.entries()]
    .map(([key, value]) => `${key}=${value}`)
    .sort()
    .join('\n');

  const secretKey = createHmac('sha256', 'WebAppData').update(botToken).digest();
  const computed = createHmac('sha256', secretKey).update(dataCheckString).digest('hex');
  return computed === hash ? params : null;
}
