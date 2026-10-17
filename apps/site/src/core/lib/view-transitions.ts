// view-transition-name of a tour's cover: the same on its card and on its
// page, so the browser flies one into the other (ui/shared-transition.ts).
export function tourCoverTransition(id: string): string {
  return `tour-cover-${id.replace(/[^a-zA-Z0-9_-]/g, '')}`;
}
