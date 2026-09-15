import { GalleryRegion } from '~/shared/api/types';

// The 7 regions (oblasts) of Kyrgyzstan, in a fixed display order.
export const KYRGYZSTAN_REGIONS = [
  GalleryRegion.CHUY,
  GalleryRegion.ISSYK_KUL,
  GalleryRegion.NARYN,
  GalleryRegion.TALAS,
  GalleryRegion.OSH,
  GalleryRegion.JALAL_ABAD,
  GalleryRegion.BATKEN,
] as const;
