import { AchievementsPage } from '@/pages/achievements/achievements.page';
import { pageTestProviders } from '@/testing/providers';
import { HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

describe('AchievementsPage', () => {
  let api: HttpTestingController;

  async function openEditor(): Promise<HTMLElement> {
    TestBed.configureTestingModule({ imports: [AchievementsPage], providers: pageTestProviders() });
    api = TestBed.inject(HttpTestingController);
    const fixture = TestBed.createComponent(AchievementsPage);
    TestBed.tick();
    api.expectOne('/achievements/admin/all').flush([]);
    api.expectOne('/achievements/admin/awards').flush([]);
    await fixture.whenStable();
    const add = [...(fixture.nativeElement as HTMLElement).querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Добавить'),
    );
    add?.click();
    await fixture.whenStable();
    // The dialog renders in the CDK overlay, outside the component.
    return document.querySelector('.cdk-overlay-container') as HTMLElement;
  }

  async function submitDialog(overlay: HTMLElement): Promise<void> {
    overlay.querySelector('form')?.dispatchEvent(new Event('submit', { cancelable: true }));
    await new Promise((resolve) => setTimeout(resolve));
    TestBed.tick();
  }

  function type(input: HTMLInputElement | HTMLTextAreaElement, value: string): void {
    input.value = value;
    input.dispatchEvent(new Event('input'));
  }

  it('requires the Russian title', async () => {
    const overlay = await openEditor();
    await submitDialog(overlay);
    api.expectNone('/achievements');
    expect(overlay.textContent).toContain('Название на русском обязательно');
  });

  it('sends only the filled-in languages', async () => {
    const overlay = await openEditor();
    const inputs = overlay.querySelectorAll<HTMLInputElement>('input.ui-input');
    type(inputs[0] as HTMLInputElement, 'Покоритель Ала-Куля');
    type(inputs[2] as HTMLInputElement, 'Ala-Kul conqueror');
    await submitDialog(overlay);
    const req = api.expectOne('/achievements');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({
      title: { ru: 'Покоритель Ала-Куля', en: 'Ala-Kul conqueror' },
      description: {},
      icon: 'i-mdi-trophy',
      rarity: 'common',
      sortOrder: 0,
      isActive: true,
    });
  });
});
