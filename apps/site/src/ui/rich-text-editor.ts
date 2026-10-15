import { UploadApi } from '@/core/api/upload';
import { APP_CONFIG } from '@/core/config/app-config';
import { resolveMediaUrl } from '@/core/lib/media-url';
import { isPlatformBrowser } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  type ElementRef,
  PLATFORM_ID,
  ViewEncapsulation,
  afterNextRender,
  effect,
  inject,
  input,
  model,
  signal,
  viewChild,
} from '@angular/core';
import type { FormValueControl } from '@angular/forms/signals';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import type { Editor } from '@tiptap/core';

interface ToolbarButton {
  // Translation key of the button's name.
  label: string;
  icon?: string;
  text?: string;
  active?: () => boolean;
  run: () => void;
}

// A Tiptap HTML editor with a preview tab; the field holds the HTML. Tiptap
// loads in the browser only (the pages using it render on the client).
@Component({
  selector: 'app-rich-text-editor',
  imports: [TranslocoPipe],
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
            [attr.aria-pressed]="mode() === 'edit'"
            (click)="mode.set('edit')"
          >
            {{ 'editor.write' | transloco }}
          </button>
          <button
            type="button"
            class="px-2.5 py-1 text-xs font-medium"
            [class]="mode() === 'preview' ? 'bg-text text-surface' : 'text-text-muted'"
            [attr.aria-pressed]="mode() === 'preview'"
            (click)="mode.set('preview')"
          >
            {{ 'editor.preview' | transloco }}
          </button>
        </div>
        @if (mode() === 'edit') {
          @for (button of toolbar; track button.label) {
            <button
              type="button"
              class="ui-btn-ghost ui-btn-xs aria-pressed:bg-text/10"
              [attr.aria-pressed]="button.active ? button.active() && revision() >= 0 : null"
              [attr.aria-label]="button.label | transloco"
              [title]="button.label | transloco"
              [disabled]="uploading()"
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
      @if (uploading()) {
        <p class="border-b border-border px-3 py-1.5 text-xs text-text-muted" role="status">
          {{ 'editor.uploadingImage' | transloco }}
        </p>
      }
      <div #host class="rte-content prose prose-sm dark:prose-invert max-w-none p-3" [class.hidden]="mode() !== 'edit'"></div>
      @if (mode() === 'preview') {
        <div class="prose prose-sm dark:prose-invert min-h-[280px] max-w-none p-3" [innerHTML]="value()"></div>
      }
    </div>
  `,
})
export class RichTextEditor implements FormValueControl<string> {
  readonly value = model('');
  readonly placeholder = input('');

  private readonly host = viewChild.required<ElementRef<HTMLElement>>('host');
  private readonly imageInput = viewChild<ElementRef<HTMLInputElement>>('imageInput');
  private readonly uploads = inject(UploadApi);
  private readonly apiBaseUrl = inject(APP_CONFIG).apiBaseUrl;
  private readonly transloco = inject(TranslocoService);
  private editor: Editor | null = null;

  protected readonly mode = signal<'edit' | 'preview'>('edit');
  protected readonly uploading = signal(false);
  // Bumped on every editor transaction so active toolbar states re-render.
  protected readonly revision = signal(0);

  protected readonly toolbar: ToolbarButton[] = [
    {
      label: 'editor.bold',
      icon: 'icon-[heroicons--bold]',
      active: () => this.is('bold'),
      run: () => this.chain()?.toggleBold().run(),
    },
    {
      label: 'editor.italic',
      icon: 'icon-[heroicons--italic]',
      active: () => this.is('italic'),
      run: () => this.chain()?.toggleItalic().run(),
    },
    {
      label: 'editor.strike',
      icon: 'icon-[heroicons--strikethrough]',
      active: () => this.is('strike'),
      run: () => this.chain()?.toggleStrike().run(),
    },
    {
      label: 'editor.heading2',
      text: 'H2',
      active: () => this.is('heading', { level: 2 }),
      run: () => this.chain()?.toggleHeading({ level: 2 }).run(),
    },
    {
      label: 'editor.heading3',
      text: 'H3',
      active: () => this.is('heading', { level: 3 }),
      run: () => this.chain()?.toggleHeading({ level: 3 }).run(),
    },
    {
      label: 'editor.bulletList',
      icon: 'icon-[heroicons--list-bullet]',
      active: () => this.is('bulletList'),
      run: () => this.chain()?.toggleBulletList().run(),
    },
    {
      label: 'editor.orderedList',
      icon: 'icon-[heroicons--numbered-list]',
      active: () => this.is('orderedList'),
      run: () => this.chain()?.toggleOrderedList().run(),
    },
    {
      label: 'editor.quote',
      icon: 'icon-[heroicons--chat-bubble-left-right]',
      active: () => this.is('blockquote'),
      run: () => this.chain()?.toggleBlockquote().run(),
    },
    {
      label: 'editor.link',
      icon: 'icon-[heroicons--link]',
      active: () => this.is('link'),
      run: () => this.editLink(),
    },
    {
      label: 'editor.image',
      icon: 'icon-[heroicons--photo]',
      run: () => this.imageInput()?.nativeElement.click(),
    },
    {
      label: 'editor.undo',
      icon: 'icon-[heroicons--arrow-uturn-left]',
      run: () => this.chain()?.undo().run(),
    },
    {
      label: 'editor.redo',
      icon: 'icon-[heroicons--arrow-uturn-right]',
      run: () => this.chain()?.redo().run(),
    },
  ];

  constructor() {
    const destroyRef = inject(DestroyRef);
    if (!isPlatformBrowser(inject(PLATFORM_ID))) return;
    // ProseMirror needs the DOM, and Tiptap is heavy: load it on render.
    afterNextRender(async () => {
      const [{ Editor }, { default: StarterKit }, { default: Image }, { default: Placeholder }] =
        await Promise.all([
          import('@tiptap/core'),
          import('@tiptap/starter-kit'),
          import('@tiptap/extension-image'),
          import('@tiptap/extension-placeholder'),
        ]);
      if (destroyRef.destroyed) return;
      this.editor = new Editor({
        element: this.host().nativeElement,
        content: this.value(),
        extensions: [
          StarterKit.configure({ link: { openOnClick: false } }),
          Image,
          Placeholder.configure({ placeholder: this.placeholder() }),
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
    destroyRef.onDestroy(() => this.editor?.destroy());
  }

  private chain(): ReturnType<Editor['chain']> | undefined {
    return this.editor?.chain().focus();
  }

  private is(name: string, attributes?: Record<string, unknown>): boolean {
    return this.editor?.isActive(name, attributes) ?? false;
  }

  // Empty input removes the link, as on the Nuxt site.
  private editLink(): void {
    const current = (this.editor?.getAttributes('link')['href'] as string | undefined) ?? '';
    const url = window.prompt(this.transloco.translate('guides.editor.linkPrompt'), current);
    if (url === null) return;
    const chain = this.chain()?.extendMarkRange('link');
    if (url.trim()) chain?.setLink({ href: url.trim() }).run();
    else chain?.unsetLink().run();
  }

  protected async onImage(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    this.uploading.set(true);
    try {
      const url = await this.uploads.uploadImage(file);
      this.chain()
        ?.setImage({ src: resolveMediaUrl(this.apiBaseUrl, url) })
        .run();
    } finally {
      this.uploading.set(false);
      input.value = '';
    }
  }
}
