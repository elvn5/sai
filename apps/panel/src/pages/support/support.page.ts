import type { SupportInfo } from '@/shared/api/types';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormField, form, submit } from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';

interface SupportForm {
  message: string;
  phone: string;
  telegramContact: string;
  email: string;
  workingHours: string;
}

// Support contacts the bot shows (services/users).
@Component({
  selector: 'app-support-page',
  imports: [FormField],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="max-w-xl">
      <h1 class="mb-4 text-xl font-semibold text-text">Поддержка</h1>
      <form class="ui-card flex flex-col gap-3 p-4" (submit)="save($event)" novalidate>
        <label class="ui-field">
          Сообщение (показывается в боте)
          <textarea class="ui-input" rows="4" [formField]="supportForm.message"></textarea>
        </label>
        <label class="ui-field">Телефон <input class="ui-input" [formField]="supportForm.phone" /></label>
        <label class="ui-field">
          Telegram-контакт
          <input class="ui-input" placeholder="@username" [formField]="supportForm.telegramContact" />
        </label>
        <label class="ui-field">Email <input class="ui-input" [formField]="supportForm.email" /></label>
        <label class="ui-field">Часы работы <input class="ui-input" [formField]="supportForm.workingHours" /></label>
        <div class="mt-2 flex items-center gap-2">
          <button type="submit" class="ui-btn-primary" [disabled]="supportForm().submitting()">Сохранить</button>
          @if (saved()) {
            <span class="text-sm text-green-600">Сохранено</span>
          }
        </div>
      </form>
    </div>
  `,
})
export class SupportPage {
  private readonly http = inject(HttpClient);
  protected readonly model = signal<SupportForm>({
    message: '',
    phone: '',
    telegramContact: '',
    email: '',
    workingHours: '',
  });
  protected readonly supportForm = form(this.model);
  protected readonly saved = signal(false);

  constructor() {
    void this.load();
  }

  private async load(): Promise<void> {
    const data = await firstValueFrom(this.http.get<SupportInfo>('/support'));
    this.model.set({
      message: data.message,
      phone: data.phone ?? '',
      telegramContact: data.telegramContact ?? '',
      email: data.email ?? '',
      workingHours: data.workingHours ?? '',
    });
  }

  protected async save(event: Event): Promise<void> {
    event.preventDefault();
    this.saved.set(false);
    await submit(this.supportForm, async () => {
      const value = this.model();
      await firstValueFrom(
        this.http.put('/support', {
          message: value.message,
          phone: value.phone || null,
          telegramContact: value.telegramContact || null,
          email: value.email || null,
          workingHours: value.workingHours || null,
        }),
      );
      this.saved.set(true);
      return undefined;
    });
  }
}
