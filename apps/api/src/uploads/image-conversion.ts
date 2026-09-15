import sharp from 'sharp';

// Cap the longest edge — gallery/cover photos are never displayed larger
// than this, and phone cameras routinely produce 4000px+ originals that add
// nothing but load time.
export const IMAGE_MAX_DIMENSION = 2000;
export const IMAGE_WEBP_QUALITY = 82;

export async function convertImageBufferToWebp(buffer: Buffer): Promise<Buffer> {
  return sharp(buffer, { animated: true })
    .rotate() // apply EXIF orientation before stripping metadata
    .resize({
      width: IMAGE_MAX_DIMENSION,
      height: IMAGE_MAX_DIMENSION,
      fit: 'inside',
      withoutEnlargement: true,
    })
    .webp({ quality: IMAGE_WEBP_QUALITY })
    .toBuffer();
}
