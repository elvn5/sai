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
