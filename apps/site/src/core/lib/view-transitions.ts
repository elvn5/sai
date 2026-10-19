// view-transition-names of the pictures that fly from a card to its page:
// the same name on both, so the browser morphs one into the other
// (ui/shared-transition.ts). Ids may hold any character; names may not.
function sharedName(kind: string, id: string): string {
  return `${kind}-${id.replace(/[^a-zA-Z0-9_-]/g, '')}`;
}

// A tour's cover.
export function tourCoverTransition(id: string): string {
  return sharedName('tour-cover', id);
}

// A hotel's cover, landing on the page's photo grid.
export function hotelCoverTransition(id: string): string {
  return sharedName('hotel-cover', id);
}

// A route's cover, landing on the page's hero.
export function trailCoverTransition(id: string): string {
  return sharedName('trail-cover', id);
}

// A local guide's portrait.
export function guideAvatarTransition(id: string): string {
  return sharedName('guide-avatar', id);
}
