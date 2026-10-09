<script setup lang="ts">
import ImageExtension from '@tiptap/extension-image';
import PlaceholderExtension from '@tiptap/extension-placeholder';
import StarterKit from '@tiptap/starter-kit';
import { EditorContent, useEditor } from '@tiptap/vue-3';
import { useApiClient } from '~/shared/api/client';

const props = defineProps<{
  modelValue: string | undefined;
  placeholder?: string;
}>();

const emit = defineEmits<{ 'update:modelValue': [string] }>();

const { t } = useI18n();
const api = useApiClient();
const imageInput = ref<HTMLInputElement | null>(null);
const uploadingImage = ref(false);

const editor = useEditor({
  content: props.modelValue ?? '',
  extensions: [
    StarterKit.configure({
      heading: { levels: [2, 3] },
      link: { openOnClick: false, autolink: true },
    }),
    ImageExtension,
    PlaceholderExtension.configure({ placeholder: props.placeholder ?? '' }),
  ],
  editorProps: {
    attributes: {
      class:
        'prose prose-neutral dark:prose-invert max-w-none min-h-[240px] rounded-b-md border border-t-0 border-border bg-surface px-3 py-2 focus:outline-none',
    },
  },
  onUpdate: ({ editor: instance }) => {
    emit('update:modelValue', instance.getHTML());
  },
});

watch(
  () => props.modelValue,
  (value: string | undefined) => {
    const html = value ?? '';
    if (editor.value && html !== editor.value.getHTML()) {
      editor.value.commands.setContent(html, { emitUpdate: false });
    }
  },
);

onBeforeUnmount(() => {
  editor.value?.destroy();
});

function toggleBold(): void {
  editor.value?.chain().focus().toggleBold().run();
}
function toggleItalic(): void {
  editor.value?.chain().focus().toggleItalic().run();
}
function toggleHeading(level: 2 | 3): void {
  editor.value?.chain().focus().toggleHeading({ level }).run();
}
function toggleBulletList(): void {
  editor.value?.chain().focus().toggleBulletList().run();
}
function toggleOrderedList(): void {
  editor.value?.chain().focus().toggleOrderedList().run();
}
function toggleBlockquote(): void {
  editor.value?.chain().focus().toggleBlockquote().run();
}

function setLink(): void {
  const previousUrl = editor.value?.getAttributes('link').href as string | undefined;
  const url = window.prompt(t('guides.editor.linkPrompt'), previousUrl ?? '');
  if (url === null) return;
  if (url === '') {
    editor.value?.chain().focus().extendMarkRange('link').unsetLink().run();
    return;
  }
  editor.value?.chain().focus().extendMarkRange('link').setLink({ href: url }).run();
}

function openImagePicker(): void {
  imageInput.value?.click();
}

async function onImageSelected(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  target.value = '';
  if (!file) return;

  uploadingImage.value = true;
  try {
    const formData = new FormData();
    formData.append('file', file);
    const upload = await api.upload<{ url: string }>('/uploads/image', formData);
    const config = useRuntimeConfig();
    editor.value
      ?.chain()
      .focus()
      .setImage({ src: `${config.public.apiBaseUrl}${upload.url}` })
      .run();
  } finally {
    uploadingImage.value = false;
  }
}
</script>

<template>
  <div>
    <div class="flex flex-wrap items-center gap-1 rounded-t-md border border-border bg-surface p-1.5">
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-mdi-format-bold"
        :class="{ 'bg-brand-500/10 text-brand-600': editor?.isActive('bold') }"
        @click="toggleBold"
      />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-mdi-format-italic"
        :class="{ 'bg-brand-500/10 text-brand-600': editor?.isActive('italic') }"
        @click="toggleItalic"
      />
      <span class="mx-1 h-5 w-px bg-border" />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-mdi-format-header-2"
        :class="{ 'bg-brand-500/10 text-brand-600': editor?.isActive('heading', { level: 2 }) }"
        @click="toggleHeading(2)"
      />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-mdi-format-header-3"
        :class="{ 'bg-brand-500/10 text-brand-600': editor?.isActive('heading', { level: 3 }) }"
        @click="toggleHeading(3)"
      />
      <span class="mx-1 h-5 w-px bg-border" />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-mdi-format-list-bulleted"
        :class="{ 'bg-brand-500/10 text-brand-600': editor?.isActive('bulletList') }"
        @click="toggleBulletList"
      />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-mdi-format-list-numbered"
        :class="{ 'bg-brand-500/10 text-brand-600': editor?.isActive('orderedList') }"
        @click="toggleOrderedList"
      />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-mdi-format-quote-close"
        :class="{ 'bg-brand-500/10 text-brand-600': editor?.isActive('blockquote') }"
        @click="toggleBlockquote"
      />
      <span class="mx-1 h-5 w-px bg-border" />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-mdi-link-variant"
        :class="{ 'bg-brand-500/10 text-brand-600': editor?.isActive('link') }"
        @click="setLink"
      />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-mdi-image-plus"
        :loading="uploadingImage"
        @click="openImagePicker"
      />
      <input ref="imageInput" type="file" accept="image/*" class="hidden" @change="onImageSelected" />
    </div>

    <EditorContent :editor="editor" />
  </div>
</template>
