import { pageTestProviders } from '@/testing/providers';
import { ParticipantFormView } from '@/ui/participant-form';
import { TestBed } from '@angular/core/testing';

function type(element: HTMLElement, selector: string, value: string): void {
  const input = element.querySelector(selector) as HTMLInputElement;
  input.value = value;
  input.dispatchEvent(new Event('input'));
}

describe('ParticipantFormView', () => {
  it('hands up the trimmed form', async () => {
    TestBed.configureTestingModule({ providers: pageTestProviders() });
    const fixture = TestBed.createComponent(ParticipantFormView);
    const saved: unknown[] = [];
    fixture.componentInstance.saved.subscribe((value) => saved.push(value));
    fixture.detectChanges();
    const element: HTMLElement = fixture.nativeElement;

    type(element, 'input[id$="-name"]', '  Мама ');
    type(element, 'input[id$="-phone"]', '+996 555 000111');
    (element.querySelector('input[value="trekking"]') as HTMLInputElement).dispatchEvent(
      new Event('change'),
    );
    type(element, 'textarea[id$="-health"]', ' астма ');
    element.querySelector('form')?.dispatchEvent(new Event('submit'));
    await fixture.whenStable();
    expect(saved).toEqual([
      {
        emergencyName: 'Мама',
        emergencyPhone: '+996 555 000111',
        experience: 'trekking',
        health: 'астма',
        notes: null,
      },
    ]);
  });

  it('refuses a form without a contact or experience', async () => {
    TestBed.configureTestingModule({ providers: pageTestProviders() });
    const fixture = TestBed.createComponent(ParticipantFormView);
    const saved: unknown[] = [];
    fixture.componentInstance.saved.subscribe((value) => saved.push(value));
    fixture.detectChanges();
    fixture.nativeElement.querySelector('form').dispatchEvent(new Event('submit'));
    await fixture.whenStable();
    fixture.detectChanges();
    expect(saved).toEqual([]);
    expect(fixture.nativeElement.textContent).toContain('participantForm.errors.emergencyName');
  });
});
