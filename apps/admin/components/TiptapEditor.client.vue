<script setup lang="ts">
import Image from '@tiptap/extension-image';
import StarterKit from '@tiptap/starter-kit';
import { EditorContent, useEditor } from '@tiptap/vue-3';

const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ 'update:modelValue': [string] }>();

const editor = useEditor({
  content: props.modelValue || '',
  immediatelyRender: false,
  extensions: [
    StarterKit.configure({
      heading: { levels: [2, 3] },
      link: {
        openOnClick: false,
        HTMLAttributes: { rel: 'noopener noreferrer', target: '_blank' },
      },
    }),
    Image.configure({ inline: false }),
  ],
  editorProps: {
    attributes: {
      class:
        'prose prose-sm max-w-none min-h-[320px] px-4 py-3 focus:outline-none dark:prose-invert',
    },
  },
  onUpdate: ({ editor }) => emit('update:modelValue', editor.getHTML()),
});

watch(
  () => props.modelValue,
  (val) => {
    if (editor.value && val !== editor.value.getHTML()) {
      editor.value.commands.setContent(val || '', { emitUpdate: false });
    }
  },
);

onBeforeUnmount(() => editor.value?.destroy());

function setLink(): void {
  const prev = editor.value?.getAttributes('link').href as string | undefined;
  const url = window.prompt('URL ссылки', prev ?? 'https://');
  if (url === null) return;
  if (url === '') {
    editor.value?.chain().focus().extendMarkRange('link').unsetLink().run();
    return;
  }
  editor.value?.chain().focus().extendMarkRange('link').setLink({ href: url }).run();
}

function addImage(): void {
  const url = window.prompt('URL изображения', 'https://');
  if (url) editor.value?.chain().focus().setImage({ src: url }).run();
}

type Btn = { icon: string; title: string; run: () => void; active?: () => boolean };
const groups = computed<Btn[][]>(() => {
  const e = editor.value;
  return [
    [
      {
        icon: 'i-lucide-bold',
        title: 'Жирный',
        run: () => e?.chain().focus().toggleBold().run(),
        active: () => !!e?.isActive('bold'),
      },
      {
        icon: 'i-lucide-italic',
        title: 'Курсив',
        run: () => e?.chain().focus().toggleItalic().run(),
        active: () => !!e?.isActive('italic'),
      },
      {
        icon: 'i-lucide-strikethrough',
        title: 'Зачёркнутый',
        run: () => e?.chain().focus().toggleStrike().run(),
        active: () => !!e?.isActive('strike'),
      },
    ],
    [
      {
        icon: 'i-lucide-heading-2',
        title: 'Заголовок H2',
        run: () => e?.chain().focus().toggleHeading({ level: 2 }).run(),
        active: () => !!e?.isActive('heading', { level: 2 }),
      },
      {
        icon: 'i-lucide-heading-3',
        title: 'Заголовок H3',
        run: () => e?.chain().focus().toggleHeading({ level: 3 }).run(),
        active: () => !!e?.isActive('heading', { level: 3 }),
      },
    ],
    [
      {
        icon: 'i-lucide-list',
        title: 'Маркированный список',
        run: () => e?.chain().focus().toggleBulletList().run(),
        active: () => !!e?.isActive('bulletList'),
      },
      {
        icon: 'i-lucide-list-ordered',
        title: 'Нумерованный список',
        run: () => e?.chain().focus().toggleOrderedList().run(),
        active: () => !!e?.isActive('orderedList'),
      },
      {
        icon: 'i-lucide-quote',
        title: 'Цитата',
        run: () => e?.chain().focus().toggleBlockquote().run(),
        active: () => !!e?.isActive('blockquote'),
      },
      {
        icon: 'i-lucide-code',
        title: 'Блок кода',
        run: () => e?.chain().focus().toggleCodeBlock().run(),
        active: () => !!e?.isActive('codeBlock'),
      },
    ],
    [
      { icon: 'i-lucide-link', title: 'Ссылка', run: setLink, active: () => !!e?.isActive('link') },
      { icon: 'i-lucide-image', title: 'Изображение по URL', run: addImage },
    ],
    [
      { icon: 'i-lucide-undo-2', title: 'Отменить', run: () => e?.chain().focus().undo().run() },
      { icon: 'i-lucide-redo-2', title: 'Повторить', run: () => e?.chain().focus().redo().run() },
    ],
  ];
});
</script>

<template>
  <div class="overflow-hidden rounded-lg border border-steel-200 dark:border-steel-800">
    <div
      class="flex flex-wrap items-center gap-1 border-b border-steel-200 bg-steel-50 p-1.5 dark:border-steel-800 dark:bg-steel-900"
    >
      <template v-for="(group, gi) in groups" :key="gi">
        <span v-if="gi > 0" class="mx-1 h-5 w-px bg-steel-200 dark:bg-steel-700" aria-hidden="true" />
        <UButton
          v-for="btn in group"
          :key="btn.title"
          :icon="btn.icon"
          size="xs"
          :color="btn.active?.() ? 'primary' : 'neutral'"
          :variant="btn.active?.() ? 'soft' : 'ghost'"
          :title="btn.title"
          :aria-label="btn.title"
          @click="btn.run"
        />
      </template>
    </div>
    <EditorContent :editor="editor" />
  </div>
</template>
