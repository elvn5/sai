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
