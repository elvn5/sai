import { pageTestProviders } from '@/testing/providers';
import {
  DateTimeField,
  calendarWeeks,
  dateKey,
  formatValue,
  keyParts,
  parseValue,
} from '@/ui/date-time-field';
import { TestBed } from '@angular/core/testing';

describe('date helpers', () => {
  it('reads and writes date and date-time values', () => {
    expect(parseValue('2026-10-24T06:30')).toEqual({
      year: 2026,
      month: 9,
      day: 24,
      hour: 6,
      minute: 30,
    });
    expect(parseValue('2026-10-24')).toEqual({ year: 2026, month: 9, day: 24, hour: 0, minute: 0 });
    expect(parseValue('')).toBeNull();
    expect(parseValue('24.10.2026')).toBeNull();
    expect(formatValue({ year: 2026, month: 0, day: 5, hour: 7, minute: 5 }, true)).toBe(
      '2026-01-05T07:05',
    );
    expect(formatValue({ year: 2026, month: 0, day: 5, hour: 7, minute: 5 }, false)).toBe(
      '2026-01-05',
    );
    expect(keyParts('2026-02-28')).toEqual({ year: 2026, month: 1, day: 28 });
  });

  it('rolls days and months over', () => {
    expect(dateKey(2026, 9, 32)).toBe('2026-11-01');
    expect(dateKey(2026, 0, 0)).toBe('2025-12-31');
    expect(dateKey(2026, 12, 1)).toBe('2027-01-01');
    expect(dateKey(2028, 1, 29)).toBe('2028-02-29');
  });

  it('lays a month out in six Monday-first weeks', () => {
    // October 2026 starts on a Thursday.
    const weeks = calendarWeeks(2026, 9);
    expect(weeks).toHaveLength(6);
    expect(weeks.every((week) => week.length === 7)).toBe(true);
    expect(weeks[0].map((day) => day.key)).toEqual([
      '2026-09-28',
      '2026-09-29',
      '2026-09-30',
      '2026-10-01',
      '2026-10-02',
      '2026-10-03',
      '2026-10-04',
    ]);
    expect(weeks[0][2].inMonth).toBe(false);
    expect(weeks[0][3].inMonth).toBe(true);
    expect(weeks[5][6].key).toBe('2026-11-08');
    // A month starting on a Monday starts the first week.
    expect(calendarWeeks(2026, 5)[0][0].key).toBe('2026-06-01');
  });
});

async function render(
  inputs: Record<string, unknown>,
  value = '',
): Promise<{
  host: HTMLElement;
  field: DateTimeField;
  stable: () => Promise<void>;
}> {
  TestBed.resetTestingModule();
  TestBed.configureTestingModule({ providers: pageTestProviders() });
  const fixture = TestBed.createComponent(DateTimeField);
  for (const [name, input] of Object.entries(inputs)) fixture.componentRef.setInput(name, input);
  fixture.componentInstance.value.set(value);
  await fixture.whenStable();
  return {
    host: fixture.nativeElement,
    field: fixture.componentInstance,
    stable: () => fixture.whenStable(),
  };
}

const trigger = (host: HTMLElement): HTMLButtonElement =>
  host.querySelector('button[aria-haspopup="dialog"]') as HTMLButtonElement;
const day = (host: HTMLElement, key: string): HTMLButtonElement =>
  host.querySelector(`[data-date="${key}"]`) as HTMLButtonElement;

describe('DateTimeField', () => {
  it('shows the value in words', async () => {
    const thisYear = new Date().getFullYear();
    const { host } = await render({}, `${thisYear}-10-24T06:00`);
    expect(trigger(host).textContent).toContain('24 октября, 06:00');
    expect(trigger(host).textContent).not.toContain(String(thisYear));

    const other = await render({ withTime: false }, '2030-06-12');
    expect(trigger(other.host).textContent).toContain('12 июня 2030');
  });

  it('picks a day and a time, keeping what was set', async () => {
    const { host, field, stable } = await render({ earliest: '2030-10-20', defaultTime: '18:00' });
    trigger(host).click();
    await stable();
    expect(host.querySelector('[role="dialog"]')).not.toBeNull();
    // The calendar opens on the earliest day when it is ahead of today.
    expect(day(host, '2030-10-20').getAttribute('tabindex')).toBe('0');
    expect(day(host, '2030-10-19').disabled).toBe(true);

    day(host, '2030-10-25').click();
    await stable();
    expect(field.value()).toBe('2030-10-25T18:00');

    const hours = host.querySelectorAll('[role="group"]')[0].querySelectorAll('button');
    const minutes = host.querySelectorAll('[role="group"]')[1].querySelectorAll('button');
    hours[7].click();
    minutes[6].click();
    await stable();
    expect(field.value()).toBe('2030-10-25T07:30');

    day(host, '2030-10-26').click();
    await stable();
    expect(field.value()).toBe('2030-10-26T07:30');
  });

  it('closes on picking a day in date-only mode, and clears', async () => {
    const touched = vi.fn();
    const { host, field, stable } = await render(
      { withTime: false, clearable: true },
      '2030-06-12',
    );
    field.touch.subscribe(touched);
    trigger(host).click();
    await stable();
    day(host, '2030-06-14').click();
    await stable();
    expect(field.value()).toBe('2030-06-14');
    expect(host.querySelector('[role="dialog"]')).toBeNull();
    expect(touched).toHaveBeenCalledTimes(1);

    trigger(host).click();
    await stable();
    (
      Array.from(host.querySelectorAll('button')).find((b) =>
        b.textContent?.includes('picker.clear'),
      ) as HTMLButtonElement
    ).click();
    await stable();
    expect(field.value()).toBe('');
  });

  it('moves through the calendar with the keyboard', async () => {
    const { host, stable } = await render({ withTime: false }, '2030-01-31');
    trigger(host).click();
    await stable();
    const press = async (key: string): Promise<void> => {
      host
        .querySelector('[data-date][tabindex="0"]')
        ?.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
      await stable();
    };
    const active = (): string | null =>
      host.querySelector('[data-date][tabindex="0"]')?.getAttribute('data-date') ?? null;

    await press('ArrowRight');
    expect(active()).toBe('2030-02-01');
    await press('ArrowUp');
    expect(active()).toBe('2030-01-25');
    await press('End');
    expect(active()).toBe('2030-01-27'); // Sunday
    await press('Home');
    expect(active()).toBe('2030-01-21'); // Monday
    await press('PageDown');
    expect(active()).toBe('2030-02-21');
  });

  it('closes on Escape and on a click outside', async () => {
    const { host, stable } = await render({}, '2030-06-12T10:00');
    trigger(host).click();
    await stable();
    host.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    await stable();
    expect(host.querySelector('[role="dialog"]')).toBeNull();

    trigger(host).click();
    await stable();
    document.body.click();
    await stable();
    expect(host.querySelector('[role="dialog"]')).toBeNull();
  });
});
