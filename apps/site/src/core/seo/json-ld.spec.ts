import { serializeJsonLd } from '@/core/seo/json-ld';

describe('serializeJsonLd', () => {
  it('cannot close its <script> early', () => {
    const json = serializeJsonLd({ name: '</script><script>alert(1)</script>' });
    expect(json).not.toContain('</script>');
    expect(JSON.parse(json).name).toBe('</script><script>alert(1)</script>');
  });
});

describe('catalog JSON-LD', () => {
  it('describes a hotel with its rating and reviews, leaving out what is unknown', async () => {
    const { lodgingBusiness } = await import('@/core/seo/json-ld');
    const data = lodgingBusiness({
      url: 'https://ermekten.com/hotels/ak-say',
      name: 'Ак-Сай',
      description: 'Гостевой дом',
      images: ['https://cdn/1.webp'],
      address: null,
      locality: 'Каракол',
      latitude: 42.49,
      longitude: 78.39,
      priceFrom: 2500,
      stars: null,
      amenities: ['Wi-Fi'],
      checkInTime: '14:00',
      checkOutTime: null,
      telephone: null,
      ratingAverage: 4.666,
      reviewCount: 3,
      reviews: [{ author: 'Aida', rating: 5, text: 'Отлично', date: '2026-09-01T10:00:00Z' }],
    });
    expect(data['address']).toEqual({
      '@type': 'PostalAddress',
      addressLocality: 'Каракол',
      addressCountry: 'KG',
    });
    expect(data['aggregateRating']).toMatchObject({ ratingValue: 4.7, reviewCount: 3 });
    expect((data['review'] as object[])[0]).toMatchObject({
      datePublished: '2026-09-01',
      reviewRating: { ratingValue: 5 },
    });
    expect(data).not.toHaveProperty('starRating');
    expect(data).not.toHaveProperty('checkoutTime');
  });

  it('gives a tour an offer in som until it starts', async () => {
    const { touristTrip } = await import('@/core/seo/json-ld');
    const trip = touristTrip({
      url: 'https://ermekten.com/tours/son-kul',
      name: 'Сон-Куль',
      description: '',
      image: null,
      startDate: '2026-11-01',
      endDate: '2026-11-03',
      place: 'Нарын',
      price: 15000,
      available: true,
      siteUrl: 'https://ermekten.com',
    });
    expect(trip['offers']).toMatchObject({
      price: 15000,
      priceCurrency: 'KGS',
      availability: 'https://schema.org/InStock',
    });
    expect(trip).not.toHaveProperty('description');
  });

  it('places a route by its start, with the summit height', async () => {
    const { touristAttraction } = await import('@/core/seo/json-ld');
    const place = touristAttraction({
      url: 'https://ermekten.com/trails/komsomolets',
      name: 'Пик Комсомолец',
      description: 'Классика Ала-Арчи',
      image: null,
      latitude: 42.5641,
      longitude: 74.4822,
      elevation: 4150,
      area: 'Ала-Арча',
    });
    expect(place).toMatchObject({
      '@type': 'TouristAttraction',
      geo: { '@type': 'GeoCoordinates', latitude: 42.5641, longitude: 74.4822, elevation: 4150 },
      containedInPlace: { name: 'Ала-Арча' },
    });
    expect(place).not.toHaveProperty('image');
    const noStart = touristAttraction({
      url: 'https://ermekten.com/trails/x',
      name: 'X',
      description: 'x',
      image: null,
      latitude: null,
      longitude: null,
      elevation: null,
      area: 'Другое',
    });
    expect(noStart).not.toHaveProperty('geo');
  });

  it('reads plain text out of HTML', async () => {
    const { plainText } = await import('@/core/seo/json-ld');
    expect(plainText('<p>Юрты&nbsp;у <b>озера</b></p><p>и баня</p>')).toBe('Юрты у озера и баня');
    expect(plainText('<p>abcdef</p>', 4)).toBe('abc…');
  });
});
