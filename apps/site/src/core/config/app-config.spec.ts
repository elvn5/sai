import { configFromEnv, contextFromEnv } from '@/core/config/app-config';

describe('configFromEnv', () => {
  it('indexes only the production domain', () => {
    expect(configFromEnv({ SITE_URL: 'https://ermekten.com/' }).indexable).toBe(true);
    expect(configFromEnv({ SITE_URL: 'https://test.ermekten.com' }).indexable).toBe(false);
  });

  it('reads the API, bot and media bucket from the environment', () => {
    const config = configFromEnv({
      SITE_URL: 'https://ermekten.com/',
      API_BASE_URL: 'https://api.ermekten.com',
      TELEGRAM_BOT_USERNAME: 'ermekten_bot',
      R2_PUBLIC_URL: 'https://media.ermekten.com/',
    });
    expect(config).toEqual({
      siteUrl: 'https://ermekten.com',
      apiBaseUrl: 'https://api.ermekten.com',
      telegramBotUsername: 'ermekten_bot',
      mediaPublicUrl: 'https://media.ermekten.com',
      indexable: true,
    });
  });

  it('keeps the internal API address out of the browser config', () => {
    const context = contextFromEnv({
      API_BASE_URL: 'https://api.ermekten.com',
      API_INTERNAL_URL: 'http://gateway:8080/',
    });
    expect(context.apiInternalUrl).toBe('http://gateway:8080');
    expect(JSON.stringify(context.config)).not.toContain('gateway');
    expect(contextFromEnv({}).apiInternalUrl).toBeUndefined();
  });
});

describe('loadClientConfig', () => {
  it('fetches the config only when the page came without server state', async () => {
    const { TransferState, makeStateKey } = await import('@angular/core');
    const { loadClientConfig } = await import('@/core/config/app-config');
    const state = new TransferState();
    const fetchJson = vi.fn(async () => ({ apiBaseUrl: 'https://api.ermekten.com' }));

    await loadClientConfig(state, fetchJson);
    expect(fetchJson).toHaveBeenCalledWith('/app-config.json');
    expect(state.get(makeStateKey<{ apiBaseUrl: string }>('app-config'), null)?.apiBaseUrl).toBe(
      'https://api.ermekten.com',
    );

    await loadClientConfig(state, fetchJson);
    expect(fetchJson).toHaveBeenCalledTimes(1);
  });
});
