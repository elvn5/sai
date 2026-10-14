import { AdminAuth } from '@/features/auth/auth';
import { LoginPage } from '@/pages/login/login.page';
import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';

describe('LoginPage', () => {
  const login = vi.fn<(password: string) => Promise<unknown>>();
  const navigateByUrl = vi.fn<(url: string) => Promise<boolean>>();

  beforeEach(() => {
    login.mockReset().mockResolvedValue({});
    navigateByUrl.mockReset().mockResolvedValue(true);
    TestBed.configureTestingModule({
      imports: [LoginPage],
      providers: [
        { provide: AdminAuth, useValue: { login } },
        { provide: Router, useValue: { navigateByUrl } },
      ],
    });
  });

  async function render(): Promise<HTMLElement> {
    const fixture = TestBed.createComponent(LoginPage);
    await fixture.whenStable();
    return fixture.nativeElement as HTMLElement;
  }

  // Submits the form and lets the async submit handler run.
  async function submitForm(el: HTMLElement): Promise<void> {
    el.querySelector('form')?.dispatchEvent(new Event('submit', { cancelable: true }));
    await new Promise((resolve) => setTimeout(resolve));
    TestBed.tick();
  }

  it('validates the password with the Zod schema before calling the API', async () => {
    const el = await render();
    await submitForm(el);
    expect(login).not.toHaveBeenCalled();
    expect(el.textContent).toContain('Введите пароль');
  });

  it('signs in and opens the panel', async () => {
    const el = await render();
    const input = el.querySelector('input') as HTMLInputElement;
    input.value = 'secret';
    input.dispatchEvent(new Event('input'));
    await submitForm(el);
    expect(login).toHaveBeenCalledWith('secret');
    expect(navigateByUrl).toHaveBeenCalledWith('/');
  });
});
