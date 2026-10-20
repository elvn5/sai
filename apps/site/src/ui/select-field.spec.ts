import { pageTestProviders } from '@/testing/providers';
import { SelectField, type SelectOption, normalize } from '@/ui/select-field';
import { TestBed } from '@angular/core/testing';

const ROUTES: SelectOption[] = [
  { value: '', label: 'Без маршрута' },
  ...[
    'Пик Учитель',
    'Кёль-Тор',
    'Барскоон',
    'Сон-Кёль',
    'Ала-Кёль',
    'Алтын-Арашан',
    'Чункурчак',
  ].map((label, i) => ({ value: `t${i}`, label })),
];

async function render(
  options: SelectOption[],
  value = '',
): Promise<{
  host: HTMLElement;
  field: SelectField;
  stable: () => Promise<void>;
}> {
  TestBed.resetTestingModule();
  TestBed.configureTestingModule({ providers: pageTestProviders() });
  const fixture = TestBed.createComponent(SelectField);
  fixture.componentRef.setInput('options', options);
  fixture.componentRef.setInput('placeholder', 'Выберите…');
  fixture.componentInstance.value.set(value);
  await fixture.whenStable();
  return {
    host: fixture.nativeElement,
    field: fixture.componentInstance,
    stable: () => fixture.whenStable(),
  };
}

const trigger = (host: HTMLElement): HTMLButtonElement =>
  host.querySelector('button') as HTMLButtonElement;
const labels = (host: HTMLElement): string[] =>
  Array.from(host.querySelectorAll('[role="option"]')).map((o) => o.textContent?.trim() ?? '');
const key = (host: HTMLElement, name: string): void => {
  host.dispatchEvent(new KeyboardEvent('keydown', { key: name, bubbles: true, cancelable: true }));
};

describe('SelectField', () => {
  it('normalizes text for search', () => {
    expect(normalize('  Кёль-Тор ')).toBe('кель-тор');
  });

  it('shows the chosen label or the placeholder', async () => {
    const { host } = await render([{ value: 'a', label: 'Альфа' }]);
    expect(trigger(host).textContent?.trim()).toBe('Выберите…');
    const chosen = await render([{ value: 'a', label: 'Альфа' }], 'a');
    expect(trigger(chosen.host).textContent?.trim()).toBe('Альфа');
  });

  it('picks with a click and closes', async () => {
    const touched = vi.fn();
    const { host, field, stable } = await render(ROUTES.slice(0, 4));
    field.touch.subscribe(touched);
    trigger(host).click();
    await stable();
    // A short list has no search box.
    expect(host.querySelector('input[type="search"]')).toBeNull();
    (host.querySelectorAll('[role="option"]')[2] as HTMLElement).click();
    await stable();
    expect(field.value()).toBe('t1');
    expect(host.querySelector('[role="listbox"]')).toBeNull();
    expect(touched).toHaveBeenCalledTimes(1);
  });

  it('searches long lists, ignoring «ё», and picks with the keyboard', async () => {
    const { host, field, stable } = await render(ROUTES);
    key(host, 'ArrowDown'); // opens
    await stable();
    const search = host.querySelector('input[type="search"]') as HTMLInputElement;
    expect(search).not.toBeNull();
    search.value = 'кель';
    search.dispatchEvent(new Event('input'));
    await stable();
    expect(labels(host)).toEqual(['Кёль-Тор', 'Сон-Кёль', 'Ала-Кёль']);

    key(host, 'ArrowDown');
    key(host, 'Enter');
    await stable();
    expect(field.value()).toBe('t3');
    expect(trigger(host).textContent?.trim()).toBe('Сон-Кёль');

    trigger(host).click();
    await stable();
    search.value = 'нет такого';
    (host.querySelector('input[type="search"]') as HTMLInputElement).value = 'нет такого';
    host.querySelector('input[type="search"]')?.dispatchEvent(new Event('input'));
    await stable();
    expect(labels(host)).toEqual([]);
    expect(host.textContent).toContain('picker.noResults');
    key(host, 'Enter');
    key(host, 'Escape');
    await stable();
    expect(field.value()).toBe('t3');
    expect(host.querySelector('[role="listbox"]')).toBeNull();
  });

  it('opens on the chosen option and jumps by first letter', async () => {
    const { host, stable } = await render(ROUTES.slice(0, 5), 't2');
    trigger(host).click();
    await stable();
    const active = (): string | undefined =>
      (host.querySelector('[data-active="true"]') as HTMLElement | null)?.textContent?.trim();
    expect(active()).toBe('Барскоон');
    key(host, 'к');
    await stable();
    expect(active()).toBe('Кёль-Тор');
    key(host, 'End');
    await stable();
    expect(active()).toBe('Сон-Кёль');
    key(host, 'Home');
    await stable();
    expect(active()).toBe('Без маршрута');
  });

  it('closes on a click outside', async () => {
    const { host, stable } = await render(ROUTES);
    trigger(host).click();
    await stable();
    document.body.click();
    await stable();
    expect(host.querySelector('[role="listbox"]')).toBeNull();
  });
});
