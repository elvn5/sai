import { isTelegramLaunch } from '@/core/telegram/telegram-web-app';

describe('isTelegramLaunch', () => {
  const empty = { getItem: () => null };

  it('recognizes Telegram launch data in the URL or the session', () => {
    expect(isTelegramLaunch('#tgWebAppData=query_id%3D1', empty)).toBe(true);
    expect(isTelegramLaunch('', { getItem: () => '{"tgWebAppData":"x"}' })).toBe(true);
  });

  it('leaves ordinary visits alone', () => {
    expect(isTelegramLaunch('', empty)).toBe(false);
    expect(isTelegramLaunch('#reviews', null)).toBe(false);
    expect(
      isTelegramLaunch('', {
        getItem: () => {
          throw new Error('blocked');
        },
      }),
    ).toBe(false);
  });
});
