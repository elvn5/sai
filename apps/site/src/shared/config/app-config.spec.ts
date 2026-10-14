import { configFromEnv } from '@/shared/config/app-config';

describe('configFromEnv', () => {
  it('indexes only the production domain', () => {
    expect(configFromEnv({ SITE_URL: 'https://ermekten.com/' }).indexable).toBe(true);
    expect(configFromEnv({ SITE_URL: 'https://test.ermekten.com' }).indexable).toBe(false);
  });

  it('reads the API and bot from the environment', () => {
    const config = configFromEnv({
      SITE_URL: 'https://ermekten.com/',
      API_BASE_URL: 'https://api.ermekten.com',
      TELEGRAM_BOT_USERNAME: 'ermekten_bot',
    });
    expect(config).toEqual({
      siteUrl: 'https://ermekten.com',
      apiBaseUrl: 'https://api.ermekten.com',
      telegramBotUsername: 'ermekten_bot',
      indexable: true,
    });
  });
});
