import type { ChatMessage, ChatThread, CrmTemplate } from '@/core/api/types';
import { ChatUnread, whileVisible } from '@/core/chat/chats';
import { injectLang } from '@/core/i18n/current-lang';
import { formatInBishkek, personName } from '@/core/lib/format';
import { FieldErrors } from '@/ui/field-errors';
import { PersonAvatar } from '@/ui/person-avatar';
import { StateBlock } from '@/ui/state-block';
import { HttpClient, httpResource } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  afterRenderEffect,
  computed,
  inject,
  input,
  signal,
  viewChild,
} from '@angular/core';
import { FormField, form, submit, validateStandardSchema } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { firstValueFrom } from 'rxjs';
import { z } from 'zod/mini';

const MAX_LENGTH = 4000;
// How often an open chat checks for new messages while visible.
const POLL_EVERY_MS = 5000;

interface DayGroup {
  day: string;
  messages: ChatMessage[];
}

// One chat between a partner and a traveler: the messages, grouped by day,
// and a box to write. New messages are fetched every few seconds while the
// page is visible; reading them marks the chat read.
@Component({
  selector: 'app-chat-thread',
  imports: [FieldErrors, FormField, PersonAvatar, StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: { class: 'flex flex-col min-h-0' },
  template: `
    @if (thread.error() && !thread.hasValue()) {
      <app-state-block icon="icon-[heroicons--chat-bubble-left-right]" tone="danger" title="chat.notFound" />
    } @else {
      @if (data(); as t) {
        <header class="flex items-center gap-3 border-b border-border px-4 py-3">
          <app-person-avatar [person]="t.peer" [size]="40" />
          <div class="min-w-0">
            <p class="truncate font-semibold text-text">{{ name(t) }}</p>
            <p class="text-xs text-text-muted">{{ (t.role === 'partner' ? 'chat.roleClient' : 'chat.roleOrganizer') | transloco }}</p>
          </div>
        </header>
      }
      <div #scroller class="flex min-h-64 flex-1 flex-col gap-4 overflow-y-auto px-4 py-4" aria-live="polite" [attr.aria-label]="'chat.messages' | transloco">
        @if (data(); as t) {
          @for (group of days(); track group.day) {
            <div class="flex flex-col gap-1.5">
              <p class="self-center rounded-full bg-text/5 px-3 py-0.5 text-xs text-text-muted">{{ group.day }}</p>
              @for (message of group.messages; track message.id) {
                <div class="flex flex-col" [class.items-end]="mine(message)" [class.items-start]="!mine(message)">
                  <p class="max-w-[85%] rounded-2xl px-3.5 py-2 text-[0.9375rem] whitespace-pre-line break-words" [class]="mine(message) ? 'rounded-br-md bg-brand-600 text-white' : 'rounded-bl-md bg-text/7 text-text'">{{ message.body }}</p>
                  <p class="mt-0.5 px-1 text-[0.6875rem] text-text-muted">
                    {{ time(message.createdAt) }}
                    @if (mine(message) && read(message, t)) {
                      · {{ 'chat.read' | transloco }}
                    }
                  </p>
                </div>
              }
            </div>
          } @empty {
            <p class="m-auto max-w-xs text-center text-sm text-text-muted">{{ (t.role === 'partner' ? 'chat.emptyPartner' : 'chat.emptyClient') | transloco }}</p>
          }
        } @else {
          <div class="m-auto size-8 animate-spin rounded-full border-2 border-brand-600 border-t-transparent motion-reduce:animate-none" role="status"><span class="sr-only">{{ 'chat.loading' | transloco }}</span></div>
        }
      </div>
      <form class="flex flex-col gap-1 border-t border-border p-3" novalidate (submit)="send($event)">
        @if (templateList().length > 0) {
          <div class="flex flex-wrap items-center gap-1.5 pb-1">
            <span class="text-xs text-text-muted">{{ 'crm.templates.insert' | transloco }}</span>
            @for (t of templateList(); track t.id) {
              <button type="button" class="ui-pill px-2.5 py-1 text-xs" (click)="useTemplate(t.body)">{{ t.title }}</button>
            }
          </div>
        }
        <div class="flex items-end gap-2">
          <label class="sr-only" for="chat-message">{{ 'chat.placeholder' | transloco }}</label>
          <textarea id="chat-message" class="ui-input max-h-40 min-h-11 flex-1 resize-none" rows="1" [placeholder]="'chat.placeholder' | transloco" [formField]="messageForm.body" (keydown.enter)="enter($event)"></textarea>
          <button type="submit" class="ui-btn-primary size-11 shrink-0 p-0" [disabled]="messageForm().submitting() || !data()" [attr.aria-label]="'chat.send' | transloco">
            <span class="icon-[heroicons--paper-airplane] size-5" aria-hidden="true"></span>
          </button>
        </div>
        <app-field-errors [state]="messageForm.body()" />
        @if (failed()) {
          <p class="ui-error" role="alert">{{ 'chat.sendError' | transloco }}</p>
        }
        <p class="ui-hint">{{ 'chat.enterHint' | transloco }}</p>
      </form>
    }
  `,
})
export class ChatThreadView {
  readonly chatId = input.required<string>();
  // The signed-in visitor (their messages go on the right).
  readonly me = input.required<string>();

  private readonly http = inject(HttpClient);
  private readonly transloco = inject(TranslocoService);
  private readonly unread = inject(ChatUnread);
  private readonly lang = injectLang();
  private readonly scroller = viewChild<ElementRef<HTMLElement>>('scroller');

  protected readonly thread = httpResource<ChatThread>(() => `/chats/${this.chatId()}/messages`);
  protected readonly data = computed(() => (this.thread.hasValue() ? this.thread.value() : null));
  protected readonly failed = signal(false);
  // The partner's saved texts, offered in chats with clients.
  private readonly templates = httpResource<CrmTemplate[]>(() =>
    this.data()?.role === 'partner' ? '/crm/templates' : undefined,
  );
  protected readonly templateList = computed(() =>
    this.templates.hasValue() ? this.templates.value() : [],
  );

  protected readonly days = computed<DayGroup[]>(() => {
    const groups: DayGroup[] = [];
    for (const message of this.data()?.messages ?? []) {
      const day = formatInBishkek(this.lang(), message.createdAt, {
        day: 'numeric',
        month: 'long',
      });
      const last = groups.at(-1);
      if (last?.day === day) last.messages.push(message);
      else groups.push({ day, messages: [message] });
    }
    return groups;
  });

  private readonly model = signal({ body: '' });
  protected readonly messageForm = form(this.model, (path) => {
    validateStandardSchema(
      path,
      z.object({
        body: z
          .string()
          .check(z.trim(), z.maxLength(MAX_LENGTH, this.transloco.translate('chat.tooLong'))),
      }),
    );
  });

  constructor() {
    whileVisible(POLL_EVERY_MS, () => this.thread.reload());
    // Reading the chat marks it read (the API does): refresh the badge.
    let lastCount = -1;
    afterRenderEffect(() => {
      const count = this.data()?.messages.length ?? 0;
      if (count === lastCount) return;
      if (lastCount === -1) this.unread.refresh();
      lastCount = count;
      const element = this.scroller()?.nativeElement;
      if (element) element.scrollTop = element.scrollHeight;
    });
  }

  protected name(t: ChatThread): string {
    return personName(t.peer);
  }

  protected mine(message: ChatMessage): boolean {
    return message.senderId === this.me();
  }

  protected read(message: ChatMessage, t: ChatThread): boolean {
    return t.peerReadAt !== null && t.peerReadAt >= message.createdAt;
  }

  protected time(iso: string): string {
    return formatInBishkek(this.lang(), iso, { hour: '2-digit', minute: '2-digit' });
  }

  protected useTemplate(body: string): void {
    this.model.update((m) => ({ body: m.body.trim() ? `${m.body.trim()}\n\n${body}` : body }));
  }

  // Enter sends, Shift+Enter starts a new line.
  protected enter(event: Event): void {
    if ((event as KeyboardEvent).shiftKey) return;
    event.preventDefault();
    void this.send(event);
  }

  protected async send(event: Event): Promise<void> {
    event.preventDefault();
    const body = this.model().body.trim();
    if (!body) return;
    this.failed.set(false);
    await submit(this.messageForm, async () => {
      try {
        const message = await firstValueFrom(
          this.http.post<ChatMessage>(`/chats/${this.chatId()}/messages`, { body }),
        );
        this.model.set({ body: '' });
        this.messageForm().reset();
        this.thread.update((t) => (t ? { ...t, messages: [...t.messages, message] } : t));
      } catch {
        this.failed.set(true);
      }
      return undefined;
    });
  }
}
