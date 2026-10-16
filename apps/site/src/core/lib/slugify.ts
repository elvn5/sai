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
  // Kyrgyz letters.
  ң: 'ng',
  ү: 'u',
  ө: 'o',
};

// A URL slug for a title (Cyrillic transliterated), matching the API's
// ^[a-z0-9]+(?:-[a-z0-9]+)*$, plus a short random suffix so two people
// picking the same title don't collide.
export function slugify(title: string, random: () => number = Math.random): string {
  const base = Array.from(title.toLowerCase(), (char) => CYRILLIC_TO_LATIN[char] ?? char)
    .join('')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 60)
    .replace(/-+$/, '');
  const suffix = random().toString(36).slice(2, 8);
  return base ? `${base}-${suffix}` : suffix;
}
