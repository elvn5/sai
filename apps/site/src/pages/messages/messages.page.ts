import type { ChatSummary } from '@/core/api/types';
import { AuthStore } from '@/core/auth/auth.store';
import { whileVisible } from '@/core/chat/chats';
import { injectLang } from '@/core/i18n/current-lang';
import { formatInBishkek, personName } from '@/core/lib/format';
import { loaded } from '@/core/lib/resource';
import { bindPageMeta } from '@/core/seo/seo.service';
import { ChatThreadView } from '@/ui/chat-thread';
import { ListSkeleton } from '@/ui/list-skeleton';
import { PersonAvatar } from '@/ui/person-avatar';
import { SiteLink } from '@/ui/site-link';
import { StateBlock } from '@/ui/state-block';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

// How often the list checks for new messages while visible.
const LIST_EVERY_MS = 15_000;

// /messages and /messages/:id (a traveler's chats with organizers), and the
// same in the partner cabinet (/partner/messages: chats with clients). The
// list sits next to the open chat on wide screens; on phones one of them
// shows at a time.
@Component({
  selector: 'app-messages-page',
  imports: [ChatThreadView, ListSkeleton, PersonAvatar, SiteLink, StateBlock, TranslocoPipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div [class]="partner() ? 'flex flex-col gap-4' : 'ui-container flex flex-col gap-4 py-6 lg:py-8'">
      <div [class.max-lg:hidden]="id()">
        <h1 class="text-2xl font-extrabold tracking-tight text-text">{{ 'chat.title' | transloco }}</h1>
        <p class="text-sm text-text-muted">{{ (partner() ? 'chat.subtitlePartner' : 'chat.subtitle') | transloco }}</p>
      </div>
      <div class="grid gap-4 lg:grid-cols-[18rem_minmax(0,1fr)]">
        <nav class="flex flex-col gap-1" [class.max-lg:hidden]="id()" [attr.aria-label]="'chat.title' | transloco">
          @if (chats.isLoading() && !chats.hasValue()) {
            <app-list-skeleton variant="rows" [count]="4" />
          }
          @for (chat of list(); track chat.id) {
            <a [appSiteLink]="base() + '/' + chat.id" class="flex items-center gap-3 rounded-card px-3 py-2.5 transition-colors" [class]="chat.id === id() ? 'bg-brand-500/10' : 'hover:bg-text/5'" [attr.aria-current]="chat.id === id() ? 'page' : null">
              <app-person-avatar [person]="chat.peer" [size]="40" />
              <span class="min-w-0 flex-1">
                <span class="flex items-baseline justify-between gap-2">
                  <span class="truncate font-medium text-text">{{ name(chat) }}</span>
                  <span class="shrink-0 text-[0.6875rem] text-text-muted">{{ when(chat.lastMessage.createdAt) }}</span>
                </span>
                <span class="flex items-center justify-between gap-2">
                  <span class="truncate text-sm text-text-muted">{{ chat.lastMessage.body }}</span>
                  @if (chat.unread > 0) {
                    <span class="ui-count">{{ chat.unread }}<span class="sr-only"> {{ 'chat.unread' | transloco }}</span></span>
                  }
                </span>
              </span>
            </a>
          } @empty {
            @if (chats.hasValue()) {
              <app-state-block icon="icon-[heroicons--chat-bubble-left-right]" title="chat.none" [text]="partner() ? 'chat.nonePartner' : 'chat.noneClient'" />
            }
          }
        </nav>
        @if (id(); as chatId) {
          <section class="ui-card flex h-[min(42rem,calc(100dvh-9rem))] flex-col overflow-hidden">
            <a [appSiteLink]="base()" class="flex items-center gap-1 border-b border-border px-4 py-2 text-sm text-text-muted hover:text-text lg:hidden">
              <span class="icon-[heroicons--arrow-left] size-4" aria-hidden="true"></span>{{ 'chat.back' | transloco }}
            </a>
            @if (auth.user(); as user) {
              <app-chat-thread class="flex-1" [chatId]="chatId" [me]="user.id" />
            }
          </section>
        } @else {
          <div class="ui-card hidden items-center justify-center p-8 text-sm text-text-muted lg:flex">{{ 'chat.pick' | transloco }}</div>
        }
      </div>
    </div>
  `,
})
export class MessagesPage {
  // The open chat (route parameter), if any.
  readonly id = input<string>();
  // 'partner' in the partner cabinet (links stay in it). Every chat shows:
  // a partner also writes as a client (an organizer to a driver).
  readonly scope = input<'partner' | 'all'>('all');

  protected readonly auth = inject(AuthStore);
  private readonly transloco = inject(TranslocoService);
  private readonly lang = injectLang();

  protected readonly partner = computed(() => this.scope() === 'partner');
  protected readonly base = computed(() => (this.partner() ? '/partner/messages' : '/messages'));
  protected readonly chats = httpResource<ChatSummary[]>(() => '/chats');
  protected readonly list = computed(() => loaded(this.chats) ?? []);

  constructor() {
    whileVisible(LIST_EVERY_MS, () => this.chats.reload());
    bindPageMeta(() => ({
      title: this.transloco.translate('chat.title'),
      description: this.transloco.translate('chat.subtitle'),
      robots: 'noindex',
    }));
  }

  protected name(chat: ChatSummary): string {
    return personName(chat.peer);
  }

  protected when(iso: string): string {
    const sameDay = new Date(iso).toDateString() === new Date().toDateString();
    return formatInBishkek(
      this.lang(),
      iso,
      sameDay ? { hour: '2-digit', minute: '2-digit' } : { day: 'numeric', month: 'short' },
    );
  }
}
