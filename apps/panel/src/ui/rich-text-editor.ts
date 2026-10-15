import { UploadApi } from '@/core/api/upload';
import { ConfigStore } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  type ElementRef,
  ViewEncapsulation,
  afterNextRender,
  effect,
  inject,
  model,
  signal,
  viewChild,
} from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';
import { Editor } from '@tiptap/core';
import Image from '@tiptap/extension-image';
import Placeholder from '@tiptap/extension-placeholder';
import StarterKit from '@tiptap/starter-kit';

interface ToolbarButton {
  label: string;
  icon?: string;
  text?: string;
  active?: () => boolean;
  run: () => void;
}

// A Tiptap HTML editor with a preview tab; the field holds the HTML.
@Component({
  selector: 'app-rich-text-editor',
  changeDetection: ChangeDetectionStrategy.OnPush,
  // ProseMirror renders its own DOM: these styles must reach it.
  encapsulation: ViewEncapsulation.None,
  styles: `
    .rte-content .ProseMirror {
      outline: none;
      min-height: 260px;
    }
    .rte-content .ProseMirror p.is-editor-empty:first-child::before {
      content: attr(data-placeholder);
      float: left;
      color: var(--app-text-muted);
      pointer-events: none;
      height: 0;
    }
  `,
  template: `
    <div class="ui-card overflow-hidden">
      <div class="flex flex-wrap items-center gap-1 border-b border-border bg-text/5 p-2">
        <div class="mr-auto flex overflow-hidden rounded-md border border-border bg-surface">
          <button
            type="button"
            class="px-2.5 py-1 text-xs font-medium"
            [class]="mode() === 'edit' ? 'bg-text text-surface' : 'text-text-muted'"
            (click)="mode.set('edit')"
          >
            Написать
          </button>
          <button
            type="button"
            class="px-2.5 py-1 text-xs font-medium"
            [class]="mode() === 'preview' ? 'bg-text text-surface' : 'text-text-muted'"
            (click)="mode.set('preview')"
          >
            Просмотр
          </button>
        </div>
        @if (mode() === 'edit') {
          @for (button of toolbar; track button.label) {
            <button
              type="button"
              class="ui-btn-ghost ui-btn-xs aria-pressed:bg-text/10"
              [attr.aria-pressed]="button.active ? button.active() && revision() >= 0 : null"
              [attr.aria-label]="button.label"
              [title]="button.label"
              (click)="button.run()"
            >
              @if (button.icon) {
                <span [class]="button.icon + ' size-4'" aria-hidden="true"></span>
              } @else {
                {{ button.text }}
              }
            </button>
          }
          <input #imageInput type="file" accept="image/*" class="hidden" (change)="onImage($event)" />
        }
      </div>
      <div #host class="rte-content prose prose-sm max-w-none p-3" [class.hidden]="mode() !== 'edit'"></div>
      @if (mode() === 'preview') {
        <div class="prose prose-sm min-h-[280px] max-w-none p-3" [innerHTML]="value()"></div>
      }
    </div>
  `,
})
export class RichTextEditor implements FormValueControl<string> {
  readonly value = model('');

  private readonly host = viewChild.required<ElementRef<HTMLElement>>('host');
  private readonly imageInput = viewChild<ElementRef<HTMLInputElement>>('imageInput');
  private readonly uploads = inject(UploadApi);
  private readonly apiBaseUrl = inject(ConfigStore).config.apiBaseUrl;
  private editor: Editor | null = null;

  protected readonly mode = signal<'edit' | 'preview'>('edit');
  // Bumped on every editor transaction so active toolbar states re-render.
  protected readonly revision = signal(0);

  protected readonly toolbar: ToolbarButton[] = [
    {
      label: 'Жирный',
      icon: 'icon-[heroicons--bold]',
      active: () => this.is('bold'),
      run: () => this.chain().toggleBold().run(),
    },
    {
      label: 'Курсив',
      icon: 'icon-[heroicons--italic]',
      active: () => this.is('italic'),
      run: () => this.chain().toggleItalic().run(),
    },
    {
      label: 'Зачёркнутый',
      icon: 'icon-[heroicons--strikethrough]',
      active: () => this.is('strike'),
      run: () => this.chain().toggleStrike().run(),
    },
    {
      label: 'Заголовок 2',
      text: 'H2',
      active: () => this.is('heading', { level: 2 }),
      run: () => this.chain().toggleHeading({ level: 2 }).run(),
    },
    {
      label: 'Заголовок 3',
      text: 'H3',
      active: () => this.is('heading', { level: 3 }),
      run: () => this.chain().toggleHeading({ level: 3 }).run(),
    },
    {
      label: 'Список',
      icon: 'icon-[heroicons--list-bullet]',
      active: () => this.is('bulletList'),
      run: () => this.chain().toggleBulletList().run(),
    },
    {
      label: 'Нумерованный список',
      icon: 'icon-[heroicons--numbered-list]',
      active: () => this.is('orderedList'),
      run: () => this.chain().toggleOrderedList().run(),
    },
    {
      label: 'Цитата',
      icon: 'icon-[heroicons--chat-bubble-left-right]',
      active: () => this.is('blockquote'),
      run: () => this.chain().toggleBlockquote().run(),
    },
    {
      label: 'Ссылка',
      icon: 'icon-[heroicons--link]',
      active: () => this.is('link'),
      run: () => this.addLink(),
    },
    {
      label: 'Изображение',
      icon: 'icon-[heroicons--photo]',
      run: () => this.imageInput()?.nativeElement.click(),
    },
    {
      label: 'Отменить',
      icon: 'icon-[heroicons--arrow-uturn-left]',
      run: () => this.chain().undo().run(),
    },
    {
      label: 'Повторить',
      icon: 'icon-[heroicons--arrow-uturn-right]',
      run: () => this.chain().redo().run(),
    },
  ];

  constructor() {
    // ProseMirror needs the DOM: create the editor in the browser only.
    afterNextRender(() => {
      this.editor = new Editor({
        element: this.host().nativeElement,
        content: this.value(),
        extensions: [
          StarterKit.configure({ link: { openOnClick: false } }),
          Image,
          Placeholder.configure({ placeholder: 'Начните писать...' }),
        ],
        onUpdate: ({ editor }) => this.value.set(editor.getHTML()),
        onTransaction: () => this.revision.update((n) => n + 1),
      });
    });
    // Outside changes (a post loaded after the editor started) replace the content.
    effect(() => {
      const html = this.value();
      if (this.editor && html !== this.editor.getHTML()) {
        this.editor.commands.setContent(html, { emitUpdate: false });
      }
    });
    inject(DestroyRef).onDestroy(() => this.editor?.destroy());
  }

  private chain(): ReturnType<Editor['chain']> {
    if (!this.editor) throw new Error('editor is not ready');
    return this.editor.chain().focus();
  }

  private is(name: string, attributes?: Record<string, unknown>): boolean {
    return this.editor?.isActive(name, attributes) ?? false;
  }

  private addLink(): void {
    const url = window.prompt('Ссылка (URL)');
    if (url) this.chain().extendMarkRange('link').setLink({ href: url }).run();
  }

  protected async onImage(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    try {
      const url = await this.uploads.uploadImage(file);
      this.chain()
        .setImage({ src: resolveMediaUrl(this.apiBaseUrl, url) })
        .run();
    } finally {
      input.value = '';
    }
  }
}
