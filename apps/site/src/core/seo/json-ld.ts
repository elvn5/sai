// schema.org JSON-LD builders (https://schema.org). Every public page gets
// Organization and WebSite (App); detail pages add BreadcrumbList and their
// own type (TouristTrip, LodgingBusiness, Article…) as their wave lands.

export type JsonLd = Record<string, unknown>;

const CONTEXT = 'https://schema.org';

export function organization(siteUrl: string, name: string): JsonLd {
  return {
    '@context': CONTEXT,
    '@type': 'Organization',
    '@id': `${siteUrl}/#organization`,
    name,
    url: siteUrl,
    logo: `${siteUrl}/logo.svg`,
  };
}

export function webSite(siteUrl: string, name: string, inLanguage: string): JsonLd {
  return {
    '@context': CONTEXT,
    '@type': 'WebSite',
    '@id': `${siteUrl}/#website`,
    name,
    url: siteUrl,
    inLanguage,
    publisher: { '@id': `${siteUrl}/#organization` },
  };
}

export interface Crumb {
  name: string;
  // Absolute URL of the crumb's page.
  url: string;
}

// The trail from the home page to this one; the last crumb is the page.
export function breadcrumbList(crumbs: Crumb[]): JsonLd {
  return {
    '@context': CONTEXT,
    '@type': 'BreadcrumbList',
    itemListElement: crumbs.map((crumb, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      name: crumb.name,
      item: crumb.url,
    })),
  };
}

// For a <script type="application/ld+json">: "</script>" inside a string
// value must not end the script early.
export function serializeJsonLd(data: JsonLd): string {
  return JSON.stringify(data).replace(/</g, '\\u003c');
}

export interface ArticleData {
  type: 'BlogPosting' | 'Article';
  url: string;
  headline: string;
  description?: string | null;
  image?: string | null;
  datePublished?: string | null;
  dateModified?: string | null;
  inLanguage: string;
  // The site itself wrote it (editorial content); partners' texts carry no
  // public author record yet.
  byOrganization: boolean;
  siteUrl: string;
}

// A blog post (BlogPosting) or guide (Article).
export function article(data: ArticleData): JsonLd {
  const organizationRef = { '@id': `${data.siteUrl}/#organization` };
  return dropEmpty({
    '@context': CONTEXT,
    '@type': data.type,
    mainEntityOfPage: data.url,
    headline: data.headline.slice(0, 110),
    description: data.description,
    image: data.image ? [data.image] : null,
    datePublished: data.datePublished,
    dateModified: data.dateModified ?? data.datePublished,
    inLanguage: data.inLanguage,
    author: data.byOrganization ? organizationRef : null,
    publisher: organizationRef,
  });
}

export interface GalleryImage {
  url: string;
  caption?: string | null;
  location?: string | null;
}

export function imageGallery(name: string, url: string, images: GalleryImage[]): JsonLd {
  return {
    '@context': CONTEXT,
    '@type': 'ImageGallery',
    name,
    url,
    image: images.map((image) =>
      dropEmpty({
        '@type': 'ImageObject',
        contentUrl: image.url,
        caption: image.caption,
        contentLocation: image.location ? { '@type': 'Place', name: image.location } : null,
      }),
    ),
  };
}

// Leaves out null, undefined and empty-string values.
function dropEmpty(data: JsonLd): JsonLd {
  return Object.fromEntries(
    Object.entries(data).filter(
      ([, value]) => value !== null && value !== undefined && value !== '',
    ),
  );
}

// Plain text from listing HTML, shortened for a description.
export function plainText(html: string, maxLength = 300): string {
  const text = html
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/\s+/g, ' ')
    .trim();
  return text.length > maxLength ? `${text.slice(0, maxLength - 1).trimEnd()}…` : text;
}

export interface RatedReview {
  author: string;
  rating: number;
  text: string;
  date: string;
}

function aggregateRating(average: number, count: number): JsonLd | null {
  return count > 0
    ? {
        '@type': 'AggregateRating',
        ratingValue: Number(average.toFixed(1)),
        reviewCount: count,
        bestRating: 5,
        worstRating: 1,
      }
    : null;
}

function reviewItems(reviews: RatedReview[]): JsonLd[] | null {
  return reviews.length > 0
    ? reviews.map((review) => ({
        '@type': 'Review',
        author: { '@type': 'Person', name: review.author },
        datePublished: review.date.slice(0, 10),
        reviewBody: review.text,
        reviewRating: {
          '@type': 'Rating',
          ratingValue: review.rating,
          bestRating: 5,
          worstRating: 1,
        },
      }))
    : null;
}

export interface LodgingData {
  url: string;
  name: string;
  description: string;
  images: string[];
  // Street address and the destination's name.
  address: string | null;
  locality: string;
  latitude: number | null;
  longitude: number | null;
  priceFrom: number | null;
  stars: number | null;
  amenities: string[];
  checkInTime: string | null;
  checkOutTime: string | null;
  telephone: string | null;
  ratingAverage: number;
  reviewCount: number;
  reviews: RatedReview[];
}

// A hotel, guesthouse or yurt camp.
export function lodgingBusiness(data: LodgingData): JsonLd {
  return dropEmpty({
    '@context': CONTEXT,
    '@type': 'LodgingBusiness',
    '@id': `${data.url}#lodging`,
    url: data.url,
    name: data.name,
    description: data.description,
    image: data.images.length > 0 ? data.images : null,
    address: dropEmpty({
      '@type': 'PostalAddress',
      streetAddress: data.address,
      addressLocality: data.locality,
      addressCountry: 'KG',
    }),
    geo:
      data.latitude !== null && data.longitude !== null
        ? { '@type': 'GeoCoordinates', latitude: data.latitude, longitude: data.longitude }
        : null,
    priceRange: data.priceFrom !== null ? `${data.priceFrom}+ KGS` : null,
    currenciesAccepted: 'KGS',
    starRating: data.stars ? { '@type': 'Rating', ratingValue: data.stars } : null,
    amenityFeature:
      data.amenities.length > 0
        ? data.amenities.map((name) => ({
            '@type': 'LocationFeatureSpecification',
            name,
            value: true,
          }))
        : null,
    checkinTime: data.checkInTime,
    checkoutTime: data.checkOutTime,
    telephone: data.telephone,
    aggregateRating: aggregateRating(data.ratingAverage, data.reviewCount),
    review: reviewItems(data.reviews),
  });
}

export interface GuideData {
  url: string;
  name: string;
  description: string;
  image: string | null;
  areas: string[];
  pricePerDay: number | null;
  telephone: string | null;
  ratingAverage: number;
  reviewCount: number;
  reviews: RatedReview[];
}

// A local guide, as the business they run (ratings are allowed there, not
// on a Person).
export function guideBusiness(data: GuideData): JsonLd {
  return dropEmpty({
    '@context': CONTEXT,
    '@type': 'LocalBusiness',
    '@id': `${data.url}#guide`,
    url: data.url,
    name: data.name,
    description: data.description,
    image: data.image,
    areaServed:
      data.areas.length > 0 ? data.areas.map((name) => ({ '@type': 'Place', name })) : null,
    address: { '@type': 'PostalAddress', addressCountry: 'KG' },
    priceRange: data.pricePerDay !== null ? `${data.pricePerDay}+ KGS` : null,
    telephone: data.telephone,
    aggregateRating: aggregateRating(data.ratingAverage, data.reviewCount),
    review: reviewItems(data.reviews),
  });
}

export interface TripData {
  url: string;
  name: string;
  description: string;
  image: string | null;
  startDate: string;
  endDate: string;
  place: string | null;
  price: number | null;
  // Still bookable (it hasn't started).
  available: boolean;
  siteUrl: string;
}

// A tour with dates and its price.
export function touristTrip(data: TripData): JsonLd {
  return dropEmpty({
    '@context': CONTEXT,
    '@type': 'TouristTrip',
    '@id': `${data.url}#trip`,
    url: data.url,
    name: data.name,
    description: data.description,
    image: data.image,
    departureTime: data.startDate,
    arrivalTime: data.endDate,
    itinerary: data.place ? { '@type': 'Place', name: data.place } : null,
    provider: { '@id': `${data.siteUrl}/#organization` },
    offers:
      data.price !== null && data.price > 0
        ? {
            '@type': 'Offer',
            url: data.url,
            price: data.price,
            priceCurrency: 'KGS',
            availability: data.available
              ? 'https://schema.org/InStock'
              : 'https://schema.org/SoldOut',
            validThrough: data.startDate,
          }
        : null,
  });
}

// The items a catalog page shows, in order.
export function itemList(items: { url: string; name: string }[]): JsonLd {
  return {
    '@context': CONTEXT,
    '@type': 'ItemList',
    itemListElement: items.map((item, index) => ({
      '@type': 'ListItem',
      position: index + 1,
      url: item.url,
      name: item.name,
    })),
  };
}

export interface AttractionData {
  url: string;
  name: string;
  description: string;
  image: string | null;
  latitude: number | null;
  longitude: number | null;
  // Height of the summit (or start), in meters.
  elevation: number | null;
  area: string;
}

// A mountain route: where it starts, and the area it's in.
export function touristAttraction(data: AttractionData): JsonLd {
  return dropEmpty({
    '@context': CONTEXT,
    '@type': 'TouristAttraction',
    '@id': `${data.url}#place`,
    url: data.url,
    name: data.name,
    description: data.description,
    image: data.image,
    isAccessibleForFree: true,
    geo:
      data.latitude !== null && data.longitude !== null
        ? dropEmpty({
            '@type': 'GeoCoordinates',
            latitude: data.latitude,
            longitude: data.longitude,
            elevation: data.elevation,
          })
        : null,
    containedInPlace: {
      '@type': 'Place',
      name: data.area,
      address: { '@type': 'PostalAddress', addressCountry: 'KG' },
    },
  });
}
