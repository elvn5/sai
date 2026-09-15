// One-off (but idempotent — safe to re-run) sweep that converts any
// pre-existing JPEG/PNG/GIF uploads to WebP and rewrites every reference to
// them across the DB (cover images, gallery photos, avatars, and inline
// <img> tags inside rich-text content). New uploads already come in as
// WebP via the /uploads/image endpoint — this only exists to catch files
// that were uploaded before that changed.
import 'reflect-metadata';
import { readFile, readdir, unlink, writeFile } from 'node:fs/promises';
import { extname, join } from 'node:path';
import { AppDataSource } from '@app/database';
import { convertImageBufferToWebp } from '../uploads/image-conversion';
import { UPLOADS_DIR } from '../uploads/uploads.constants';

const CONVERTIBLE_EXTENSIONS = new Set(['.jpg', '.jpeg', '.png', '.gif']);

async function updateReferences(oldUrl: string, newUrl: string): Promise<void> {
  const tables: Array<{ table: string; urlColumn: string; contentColumn?: string }> = [
    { table: 'gallery_items', urlColumn: 'imageUrl' },
    { table: 'guides', urlColumn: 'coverImageUrl', contentColumn: 'content' },
    { table: 'tours', urlColumn: 'coverImageUrl', contentColumn: 'content' },
    { table: 'blog_posts', urlColumn: 'coverImageUrl', contentColumn: 'content' },
    { table: 'page_seo', urlColumn: 'ogImageUrl' },
    { table: 'users', urlColumn: 'avatarUrl' },
  ];

  for (const { table, urlColumn, contentColumn } of tables) {
    await AppDataSource.query(
      `UPDATE "${table}" SET "${urlColumn}" = $2 WHERE "${urlColumn}" = $1`,
      [oldUrl, newUrl],
    );
    if (contentColumn) {
      await AppDataSource.query(
        `UPDATE "${table}" SET "${contentColumn}" = REPLACE("${contentColumn}", $1, $2) WHERE "${contentColumn}" LIKE ('%' || $1 || '%')`,
        [oldUrl, newUrl],
      );
    }
  }
}

async function main(): Promise<void> {
  await AppDataSource.initialize();

  const files = await readdir(UPLOADS_DIR);
  const targets = files.filter((name) => CONVERTIBLE_EXTENSIONS.has(extname(name).toLowerCase()));
  console.log(`Found ${targets.length} image(s) to convert out of ${files.length} file(s) total.`);

  let converted = 0;
  let failed = 0;

  for (const filename of targets) {
    const oldUrl = `/uploads/${filename}`;
    const newFilename = `${filename.slice(0, -extname(filename).length)}.webp`;
    const newUrl = `/uploads/${newFilename}`;

    try {
      const buffer = await readFile(join(UPLOADS_DIR, filename));
      const webp = await convertImageBufferToWebp(buffer);
      await writeFile(join(UPLOADS_DIR, newFilename), webp);
      await updateReferences(oldUrl, newUrl);
      await unlink(join(UPLOADS_DIR, filename));
      converted += 1;
      console.log(`Converted ${filename} -> ${newFilename}`);
    } catch (error) {
      failed += 1;
      console.error(`Failed to convert ${filename}:`, error);
    }
  }

  console.log(`Done. Converted ${converted}, failed ${failed}.`);
  await AppDataSource.destroy();
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
