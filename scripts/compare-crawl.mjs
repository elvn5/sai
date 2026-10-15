#!/usr/bin/env node
// Compares what search engines see on the Nuxt site and on the Angular site
// for the sections of a migration wave, before nginx switches them
// (docs/angular-migration-plan.md, section 5; docs/strangler-nginx.md).
//
//   node scripts/compare-crawl.mjs --old https://ermekten.com \
//     --new http://127.0.0.1:3005 --host ermekten.com \
//     --sections /blog,/guides,/gallery,/hall-of-fame
//
// URLs come from the Angular site's sitemap, filtered to the sections (both
// sites read the same API). For every page it fetches both sides and
// compares the HTTP status, title, description, canonical, robots, h1,
// hreflang links and JSON-LD types. Exit code 1 when a page differs in
// status, canonical, title, h1 or indexability; new hreflang links or
// JSON-LD on the Angular side are reported as additions, not failures.
// --host sends that Host header to --new (the Angular server only answers
// for its allowed hosts, e.g. when reached on 127.0.0.1:SITE_PORT).
// hreflang is compared by language ("ru" = "ru-RU"); a language the new
// side drops fails only if its target page exists there (Nuxt links every
// language even for content written in one, whose slug answers 404).
// --ignore-robots skips indexability (a Nuxt dev server is always noindex).

const args = Object.fromEntries(
  process.argv
    .slice(2)
    .join(' ')
    .split(/\s*--/)
    .filter(Boolean)
    .map((pair) => {
      const [key, ...rest] = pair.split(/[\s=]+/);
      return [key, rest.join(' ')];
    }),
);

if (!args.old || !args.new || !args.sections) {
  process.stderr.write(
    'usage: compare-crawl.mjs --old <nuxt origin> --new <angular origin> --sections /blog,/guides [--host ermekten.com] [--limit 200]\n',
  );
  process.exit(2);
}

const OLD = args.old.replace(/\/$/, '');
const NEW = args.new.replace(/\/$/, '');
const SECTIONS = args.sections.split(',').map((section) => section.trim());
const LIMIT = Number(args.limit || 500);
const SITEMAP_TYPES = { '/blog': ['blog', 'pages'], '/guides': ['guides', 'pages'] };

async function get(base, path, host) {
  const response = await fetch(`${base}${path}`, {
    redirect: 'manual',
    headers: host ? { host } : {},
    signal: AbortSignal.timeout(20_000),
  });
  return { status: response.status, body: await response.text() };
}

function decode(value) {
  return value
    .replace(/&amp;/g, '&')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .trim();
}

function attr(tag, name) {
  const match = tag.match(new RegExp(`\\s${name}="([^"]*)"`, 'i'));
  return match ? decode(match[1]) : null;
}

function tags(html, tag) {
  return html.match(new RegExp(`<${tag}\\b[^>]*>`, 'gi')) ?? [];
}

// What search engines read from a page.
export function extract(html) {
  const head = html.split(/<\/head>/i)[0] ?? html;
  const meta = (key, value) =>
    tags(head, 'meta')
      .filter((tag) => attr(tag, key) === value)
      .map((tag) => attr(tag, 'content'))
      .at(-1) ?? null;
  const links = tags(head, 'link');
  const canonical = links.find((tag) => attr(tag, 'rel') === 'canonical');
  const hreflang = Object.fromEntries(
    links
      .filter((tag) => attr(tag, 'rel') === 'alternate' && attr(tag, 'hreflang'))
      .map((tag) => [baseLanguage(attr(tag, 'hreflang')), attr(tag, 'href')]),
  );
  const jsonLd = [
    ...html.matchAll(/<script[^>]*application\/ld\+json[^>]*>([\s\S]*?)<\/script>/gi),
  ].flatMap(([, json]) => {
    try {
      const data = JSON.parse(json);
      return (Array.isArray(data) ? data : [data]).flatMap((item) => item['@type']);
    } catch {
      return ['(invalid JSON)'];
    }
  });
  const h1 = html.match(/<h1\b[^>]*>([\s\S]*?)<\/h1>/i);
  return {
    title: decode(head.match(/<title[^>]*>([\s\S]*?)<\/title>/i)?.[1] ?? ''),
    description: meta('name', 'description'),
    robots: meta('name', 'robots'),
    canonical: canonical ? attr(canonical, 'href') : null,
    hreflang,
    jsonLd: [...new Set(jsonLd)].sort(),
    h1: h1 ? decode(h1[1].replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ')) : null,
  };
}

// "ru-RU" → "ru"; "x-default" stays whole.
function baseLanguage(tag) {
  const lower = tag.toLowerCase();
  return lower === 'x-default' ? lower : lower.split('-')[0];
}

function noindex(robots) {
  return /noindex/i.test(robots ?? '');
}

async function sitemapPaths() {
  const types = new Set(SECTIONS.flatMap((section) => SITEMAP_TYPES[section] ?? ['pages']));
  const paths = new Set();
  for (const type of types) {
    const { status, body } = await get(NEW, `/sitemaps/${type}.xml`, args.host);
    if (status !== 200) throw new Error(`sitemap ${type}: HTTP ${status}`);
    for (const [, loc] of body.matchAll(/<loc>([^<]+)<\/loc>/g)) {
      const path = new URL(decode(loc)).pathname;
      const unprefixed = path.replace(/^\/(ky|en|de)(?=\/|$)/, '') || '/';
      if (
        SECTIONS.some((section) => unprefixed === section || unprefixed.startsWith(`${section}/`))
      ) {
        paths.add(path);
      }
    }
  }
  return [...paths].sort().slice(0, LIMIT);
}

async function main() {
  const paths = await sitemapPaths();
  let failures = 0;
  process.stdout.write(`Comparing ${paths.length} pages: ${OLD} (old) vs ${NEW} (new)\n\n`);

  for (const path of paths) {
    const [oldPage, newPage] = await Promise.all([get(OLD, path), get(NEW, path, args.host)]);
    const before = extract(oldPage.body);
    const after = extract(newPage.body);
    const problems = [];
    const notes = [];

    if (oldPage.status !== newPage.status)
      problems.push(`status ${oldPage.status} → ${newPage.status}`);
    if (before.title !== after.title) problems.push(`title "${before.title}" → "${after.title}"`);
    if (before.h1 !== after.h1) problems.push(`h1 "${before.h1}" → "${after.h1}"`);
    if (!('ignore-robots' in args) && noindex(before.robots) !== noindex(after.robots)) {
      problems.push(`robots "${before.robots}" → "${after.robots}"`);
    }
    if (before.canonical && before.canonical !== after.canonical) {
      problems.push(`canonical ${before.canonical} → ${after.canonical}`);
    } else if (!before.canonical && after.canonical) {
      notes.push('adds canonical');
    }
    if (before.description !== after.description)
      notes.push(`description "${before.description}" → "${after.description}"`);

    for (const [lang, href] of Object.entries(before.hreflang)) {
      if (after.hreflang[lang] === href) continue;
      if (after.hreflang[lang]) {
        problems.push(`hreflang ${lang} ${href} → ${after.hreflang[lang]}`);
        continue;
      }
      if (new URL(href).pathname === path) {
        notes.push(`drops hreflang ${lang} pointing at the page itself`);
        continue;
      }
      const target = await get(NEW, new URL(href).pathname, args.host);
      if (target.status === 200) problems.push(`drops hreflang ${lang} → ${href}`);
      else notes.push(`drops hreflang ${lang} (its target answers ${target.status})`);
    }
    const addedLangs = Object.keys(after.hreflang).filter((lang) => !(lang in before.hreflang));
    if (addedLangs.length > 0) notes.push(`adds hreflang ${addedLangs.join(', ')}`);

    const missing = before.jsonLd.filter((type) => !after.jsonLd.includes(type));
    const added = after.jsonLd.filter((type) => !before.jsonLd.includes(type));
    if (missing.length > 0) problems.push(`JSON-LD drops ${missing.join(', ')}`);
    if (added.length > 0) notes.push(`adds JSON-LD ${added.join(', ')}`);

    if (problems.length > 0) failures++;
    const mark = problems.length > 0 ? '✗' : '✓';
    process.stdout.write(`${mark} ${path}\n`);
    for (const problem of problems) process.stdout.write(`    ✗ ${problem}\n`);
    for (const note of notes) process.stdout.write(`    + ${note}\n`);
  }

  process.stdout.write(`\n${paths.length - failures} of ${paths.length} pages match.\n`);
  process.exit(failures > 0 ? 1 : 0);
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exit(2);
  });
}
