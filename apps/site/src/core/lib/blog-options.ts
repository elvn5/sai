// A blog post's badges. Mirrors services/content (migration 00004) — keep
// the two in sync. Labels are translation keys under blog.difficulty.* and
// blog.season.*; icons are static CSS classes (written out in full so
// Tailwind generates them), `anim` the motion.css keyframes they play.

export interface BadgeOption {
  value: string;
  icon: string;
  // The icon's motion: an icon-anim-* class on hover, its keyframes on entry.
  anim: 'pop' | 'spin' | 'wiggle' | 'flip' | 'turn';
  // A ui-badge-* color.
  tone: 'soft' | 'success' | 'warning' | 'danger' | 'info';
}

export const BLOG_DIFFICULTIES: BadgeOption[] = [
  { value: 'any', icon: 'icon-[mdi--hiking]', anim: 'wiggle', tone: 'info' },
  { value: 'easy', icon: 'icon-[mdi--signal-cellular-1]', anim: 'pop', tone: 'success' },
  { value: 'moderate', icon: 'icon-[mdi--signal-cellular-2]', anim: 'pop', tone: 'warning' },
  { value: 'hard', icon: 'icon-[mdi--signal-cellular-3]', anim: 'pop', tone: 'danger' },
  { value: 'expert', icon: 'icon-[mdi--summit]', anim: 'pop', tone: 'danger' },
];

export const BLOG_DURATION: Omit<BadgeOption, 'value'> = {
  icon: 'icon-[mdi--timer-sand]',
  anim: 'turn',
  tone: 'soft',
};

export const BLOG_SEASONS: BadgeOption[] = [
  { value: 'winter', icon: 'icon-[mdi--snowflake]', anim: 'spin', tone: 'info' },
  { value: 'spring', icon: 'icon-[mdi--flower-tulip-outline]', anim: 'wiggle', tone: 'success' },
  { value: 'summer', icon: 'icon-[mdi--white-balance-sunny]', anim: 'spin', tone: 'warning' },
  { value: 'autumn', icon: 'icon-[mdi--leaf-maple]', anim: 'wiggle', tone: 'warning' },
  { value: 'all_year', icon: 'icon-[mdi--calendar-refresh]', anim: 'flip', tone: 'soft' },
];

// The duration's limit in the API.
export const BLOG_DURATION_MAX = 40;
