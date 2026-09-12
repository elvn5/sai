<template>
  <div class="overflow-hidden rounded-md border border-gray-200">
    <div v-if="editor" class="flex flex-wrap gap-1 border-b border-gray-200 bg-gray-50 p-2">
      <UButton
        size="xs"
        color="gray"
        :variant="editor.isActive('bold') ? 'solid' : 'ghost'"
        icon="i-heroicons-bold"
        @click="editor.chain().focus().toggleBold().run()"
      />
      <UButton
        size="xs"
        color="gray"
        :variant="editor.isActive('italic') ? 'solid' : 'ghost'"
        icon="i-heroicons-italic"
        @click="editor.chain().focus().toggleItalic().run()"
      />
      <UButton
        size="xs"
        color="gray"
        :variant="editor.isActive('strike') ? 'solid' : 'ghost'"
        icon="i-heroicons-strikethrough"
        @click="editor.chain().focus().toggleStrike().run()"
      />
      <UButton
        size="xs"
        color="gray"
        :variant="editor.isActive('heading', { level: 2 }) ? 'solid' : 'ghost'"
        @click="editor.chain().focus().toggleHeading({ level: 2 }).run()"
      >
        H2
      </UButton>
      <UButton
        size="xs"
        color="gray"
        :variant="editor.isActive('heading', { level: 3 }) ? 'solid' : 'ghost'"
        @click="editor.chain().focus().toggleHeading({ level: 3 }).run()"
      >
        H3
      </UButton>
      <UButton
        size="xs"
        color="gray"
        :variant="editor.isActive('bulletList') ? 'solid' : 'ghost'"
        icon="i-heroicons-list-bullet"
        @click="editor.chain().focus().toggleBulletList().run()"
      />
      <UButton
        size="xs"
        color="gray"
        :variant="editor.isActive('orderedList') ? 'solid' : 'ghost'"
        icon="i-heroicons-numbered-list"
        @click="editor.chain().focus().toggleOrderedList().run()"
      />
      <UButton
        size="xs"
        color="gray"
        :variant="editor.isActive('blockquote') ? 'solid' : 'ghost'"
        icon="i-heroicons-chat-bubble-left-right"
        @click="editor.chain().focus().toggleBlockquote().run()"
      />
      <UButton
        size="xs"
        color="gray"
        :variant="editor.isActive('link') ? 'solid' : 'ghost'"
        icon="i-heroicons-link"
        @click="addLink"
      />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-heroicons-photo"
        :loading="uploadingImage"
        @click="triggerImageUpload"
      />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-heroicons-arrow-uturn-left"
        @click="editor.chain().focus().undo().run()"
      />
      <UButton
        size="xs"
        color="gray"
        variant="ghost"
        icon="i-heroicons-arrow-uturn-right"
        @click="editor.chain().focus().redo().run()"
      />
      <input
        ref="fileInput"
        type="file"
        accept="image/*"
        class="hidden"
        @change="onImageSelected"
      />
    </div>
    <EditorContent :editor="editor" class="editor-content min-h-[280px] p-3" />
  </div>
</template>

<script setup lang="ts">
import Image from '@tiptap/extension-image';
import Placeholder from '@tiptap/extension-placeholder';
import StarterKit from '@tiptap/starter-kit';
import { EditorContent, useEditor } from '@tiptap/vue-3';
import { ref, watch } from 'vue';
import { useApiClient } from '~/shared/api/client';

const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ 'update:modelValue': [value: string] }>();

const api = useApiClient();
const fileInput = ref<HTMLInputElement | null>(null);
const uploadingImage = ref(false);

const editor = useEditor({
  content: props.modelValue,
  extensions: [
    StarterKit.configure({ link: { openOnClick: false } }),
    Image,
    Placeholder.configure({ placeholder: 'Начните писать...' }),
  ],
  onUpdate: ({ editor }) => {
    emit('update:modelValue', editor.getHTML());
  },
});

watch(
  () => props.modelValue,
  (value) => {
    if (editor.value && value !== editor.value.getHTML()) {
      editor.value.commands.setContent(value, { emitUpdate: false });
    }
  },
);

function addLink(): void {
  const url = window.prompt('Ссылка (URL)');
  if (url) {
    editor.value?.chain().focus().extendMarkRange('link').setLink({ href: url }).run();
  }
}

function triggerImageUpload(): void {
  fileInput.value?.click();
}

async function onImageSelected(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  uploadingImage.value = true;
  try {
    const { url } = await api.upload<{ url: string }>('/uploads/image', file);
    const config = useRuntimeConfig();
    editor.value
      ?.chain()
      .focus()
      .setImage({ src: `${config.public.apiBaseUrl}${url}` })
      .run();
  } finally {
    uploadingImage.value = false;
    target.value = '';
  }
}
</script>

<style>
.editor-content .ProseMirror {
  outline: none;
  min-height: 260px;
}
.editor-content .ProseMirror h2 {
  font-size: 1.5rem;
  font-weight: 700;
  margin: 1rem 0 0.5rem;
}
.editor-content .ProseMirror h3 {
  font-size: 1.25rem;
  font-weight: 700;
  margin: 1rem 0 0.5rem;
}
.editor-content .ProseMirror p {
  margin: 0.5rem 0;
}
.editor-content .ProseMirror ul,
.editor-content .ProseMirror ol {
  padding-left: 1.5rem;
  margin: 0.5rem 0;
}
.editor-content .ProseMirror ul {
  list-style: disc;
}
.editor-content .ProseMirror ol {
  list-style: decimal;
}
.editor-content .ProseMirror blockquote {
  border-left: 3px solid #d1d5db;
  padding-left: 1rem;
  color: #6b7280;
  margin: 0.5rem 0;
}
.editor-content .ProseMirror img {
  max-width: 100%;
  border-radius: 0.375rem;
}
.editor-content .ProseMirror a {
  color: #0d9488;
  text-decoration: underline;
}
.editor-content .ProseMirror p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  float: left;
  color: #9ca3af;
  pointer-events: none;
  height: 0;
}
</style>
