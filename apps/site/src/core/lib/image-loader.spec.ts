import { thumbnailUrl } from '@/core/lib/image-loader';

const config = {
  apiBaseUrl: 'https://api.ermekten.com',
  mediaPublicUrl: 'https://media.ermekten.com',
};

describe('thumbnailUrl', () => {
  it('serves uploads as thumbnails at the next width up', () => {
    expect(thumbnailUrl(config, 'https://api.ermekten.com/uploads/a.webp', 600)).toBe(
      'https://api.ermekten.com/images/640/uploads/a.webp',
    );
    expect(thumbnailUrl(config, '/uploads/a.webp', 112)).toBe(
      'https://api.ermekten.com/images/128/uploads/a.webp',
    );
    expect(thumbnailUrl(config, 'https://media.ermekten.com/gallery/b.jpg', 3840)).toBe(
      'https://api.ermekten.com/images/1920/r2/gallery/b.jpg',
    );
  });

  it('leaves other images and width-less requests alone', () => {
    expect(thumbnailUrl(config, 'https://t.me/i/userpic/320/x.jpg', 640)).toBe(
      'https://t.me/i/userpic/320/x.jpg',
    );
    expect(thumbnailUrl(config, '/logo.svg', 64)).toBe('/logo.svg');
    expect(thumbnailUrl(config, '/uploads/a.webp')).toBe('/uploads/a.webp');
    expect(thumbnailUrl(config, '/uploads/a.webp?v=2', 640)).toBe('/uploads/a.webp?v=2');
    expect(
      thumbnailUrl({ ...config, mediaPublicUrl: '' }, 'https://media.ermekten.com/b.jpg', 640),
    ).toBe('https://media.ermekten.com/b.jpg');
  });
});
