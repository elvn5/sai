import { serializeJsonLd } from '@/core/seo/json-ld';

describe('serializeJsonLd', () => {
  it('cannot close its <script> early', () => {
    const json = serializeJsonLd({ name: '</script><script>alert(1)</script>' });
    expect(json).not.toContain('</script>');
    expect(JSON.parse(json).name).toBe('</script><script>alert(1)</script>');
  });
});
