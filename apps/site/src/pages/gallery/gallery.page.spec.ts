import { galleryQuery } from '@/pages/gallery/gallery.page';

describe('galleryQuery', () => {
  it('links pages and filters, leaving defaults out', () => {
    expect(galleryQuery(1, { location: null, search: null })).toBe('/gallery');
    expect(galleryQuery(3, { location: null, search: null })).toBe('/gallery?page=3');
    expect(galleryQuery(2, { location: 'Каракол', search: 'озеро' })).toBe(
      '/gallery?location=%D0%9A%D0%B0%D1%80%D0%B0%D0%BA%D0%BE%D0%BB&search=%D0%BE%D0%B7%D0%B5%D1%80%D0%BE&page=2',
    );
  });
});
