import { Icon } from '@/ui/icon';
import { TestBed } from '@angular/core/testing';

describe('Icon', () => {
  it('turns the stored Nuxt Icon name into an Iconify name', async () => {
    const fixture = TestBed.createComponent(Icon);
    fixture.componentRef.setInput('name', 'i-mdi-map-marker-path');
    await fixture.whenStable();
    const el = (fixture.nativeElement as HTMLElement).querySelector('iconify-icon');
    expect(el?.getAttribute('icon')).toBe('mdi:map-marker-path');
  });
});
