const CYRILLIC_TO_LATIN: Record<string, string> = {
  а: 'a',
  б: 'b',
  в: 'v',
  г: 'g',
  д: 'd',
  е: 'e',
  ё: 'e',
  ж: 'zh',
  з: 'z',
  и: 'i',
  й: 'y',
  к: 'k',
  л: 'l',
  м: 'm',
  н: 'n',
  о: 'o',
  п: 'p',
  р: 'r',
  с: 's',
  т: 't',
  у: 'u',
  ф: 'f',
  х: 'h',
  ц: 'ts',
  ч: 'ch',
  ш: 'sh',
  щ: 'sch',
  ъ: '',
  ы: 'y',
  ь: '',
  э: 'e',
  ю: 'yu',
  я: 'ya',
  // Kyrgyz-specific letters
  ң: 'ng',
  ү: 'u',
  ө: 'o',
};

function transliterate(value: string): string {
  return value
    .toLowerCase()
    .split('')
    .map((char) => CYRILLIC_TO_LATIN[char] ?? char)
    .join('');
}

// Generates a URL-safe slug from a title, matching the API's
// ^[a-z0-9]+(?:-[a-z0-9]+)*$ pattern, plus a short random suffix so two
// users picking the same title don't collide.
export function slugify(title: string): string {
  const base = transliterate(title)
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60);

  const suffix = Math.random().toString(36).slice(2, 8);
  return base ? `${base}-${suffix}` : suffix;
}
