import {
  PHONE_PATTERN,
  TELEGRAM_PATTERN,
  optionalPattern,
  optionalUrl,
  telegramUsername,
} from '@/core/lib/listing-form';

describe('listing form rules', () => {
  it('accepts empty or valid contacts', () => {
    const phone = optionalPattern(PHONE_PATTERN, 'bad phone');
    expect(phone.safeParse('').success).toBe(true);
    expect(phone.safeParse('+996 555 123-456').success).toBe(true);
    expect(phone.safeParse('call me').success).toBe(false);
    expect(optionalPattern(TELEGRAM_PATTERN, 'bad').safeParse('@ak_say_kg').success).toBe(true);
  });

  it('accepts only http(s) websites', () => {
    const url = optionalUrl('bad url');
    expect(url.safeParse('https://ak-say.kg').success).toBe(true);
    expect(url.safeParse('javascript:alert(1)').success).toBe(false);
    expect(url.safeParse('').success).toBe(true);
  });

  it('stores Telegram usernames without the @', () => {
    expect(telegramUsername(' @ak_say ')).toBe('ak_say');
    expect(telegramUsername('')).toBeNull();
  });
});
