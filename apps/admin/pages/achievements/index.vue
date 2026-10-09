<script setup lang="ts">
import { ApiError, useApiClient } from '~/shared/api/client';
import {
  type Achievement,
  AchievementRarity,
  type Award,
  type LocalizedText,
} from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const achievements = ref<Achievement[]>([]);
const awards = ref<Award[]>([]);
const loading = ref(false);
const pageError = ref<string | null>(null);

const LOCALES = [
  { key: 'ru', label: 'Русский (обязательно)' },
  { key: 'ky', label: 'Кыргызча' },
  { key: 'en', label: 'English' },
  { key: 'de', label: 'Deutsch' },
] as const;

const RARITIES = [
  { value: AchievementRarity.COMMON, label: 'Обычное · 10 XP' },
  { value: AchievementRarity.RARE, label: 'Редкое · 25 XP' },
  { value: AchievementRarity.EPIC, label: 'Эпическое · 50 XP' },
  { value: AchievementRarity.LEGENDARY, label: 'Легендарное · 100 XP' },
];

const RARITY_BADGE: Record<AchievementRarity, string> = {
  common: 'bg-amber-100 text-amber-800',
  rare: 'bg-sky-100 text-sky-800',
  epic: 'bg-violet-100 text-violet-800',
  legendary: 'bg-yellow-200 text-yellow-900',
};

const ICON_IDEAS = [
  'i-mdi-trophy',
  'i-mdi-medal',
  'i-mdi-image-filter-hdr',
  'i-mdi-hiking',
  'i-mdi-tent',
  'i-mdi-horse',
  'i-mdi-ski',
  'i-mdi-waves',
  'i-mdi-campfire',
  'i-mdi-camera',
  'i-mdi-star-shooting',
  'i-mdi-crown',
  'i-mdi-compass',
  'i-mdi-map-marker-path',
  'i-mdi-snowflake',
  'i-mdi-heart',
];

interface FormState {
  id: string | null;
  title: Record<string, string>;
  description: Record<string, string>;
  icon: string;
  rarity: AchievementRarity;
  sortOrder: number;
  isActive: boolean;
}

function emptyForm(): FormState {
  return {
    id: null,
    title: { ru: '', ky: '', en: '', de: '' },
    description: { ru: '', ky: '', en: '', de: '' },
    icon: 'i-mdi-trophy',
    rarity: AchievementRarity.COMMON,
    sortOrder: 0,
    isActive: true,
  };
}

const modalOpen = ref(false);
const form = ref<FormState>(emptyForm());
const saving = ref(false);
const formError = ref<string | null>(null);

async function load(): Promise<void> {
  loading.value = true;
  try {
    [achievements.value, awards.value] = await Promise.all([
      api.get<Achievement[]>('/achievements/admin/all'),
      api.get<Award[]>('/achievements/admin/awards'),
    ]);
  } finally {
    loading.value = false;
  }
}

function openCreate(): void {
  form.value = emptyForm();
  formError.value = null;
  modalOpen.value = true;
}

function openEdit(a: Achievement): void {
  form.value = {
    id: a.id,
    title: { ru: '', ky: '', en: '', de: '', ...a.title },
    description: { ru: '', ky: '', en: '', de: '', ...a.description },
    icon: a.icon,
    rarity: a.rarity,
    sortOrder: a.sortOrder,
    isActive: a.isActive,
  };
  formError.value = null;
  modalOpen.value = true;
}

function compact(text: Record<string, string>): LocalizedText {
  return Object.fromEntries(Object.entries(text).filter(([, v]) => v.trim() !== ''));
}

async function save(): Promise<void> {
  formError.value = null;
  if (!form.value.title.ru?.trim()) {
    formError.value = 'Название на русском обязательно';
    return;
  }
  saving.value = true;
  const body = {
    title: compact(form.value.title),
    description: compact(form.value.description),
    icon: form.value.icon.trim(),
    rarity: form.value.rarity,
    sortOrder: Number(form.value.sortOrder) || 0,
    isActive: form.value.isActive,
  };
  try {
    if (form.value.id) {
      await api.put(`/achievements/${form.value.id}`, body);
    } else {
      await api.post('/achievements', body);
    }
    modalOpen.value = false;
    await load();
  } catch (error) {
    formError.value = error instanceof ApiError ? error.message : 'Не удалось сохранить';
  } finally {
    saving.value = false;
  }
}

async function remove(a: Achievement): Promise<void> {
  if (!window.confirm(`Удалить достижение «${a.title.ru}»?`)) return;
  pageError.value = null;
  try {
    await api.delete(`/achievements/${a.id}`);
    await load();
  } catch (error) {
    pageError.value =
      error instanceof ApiError && error.status === 409
        ? `«${a.title.ru}» уже вручали — его можно только скрыть (снимите «Активно»).`
        : 'Не удалось удалить';
  }
}

async function revoke(award: Award): Promise<void> {
  const who = award.user.firstName ?? award.user.username ?? 'пользователя';
  if (!window.confirm(`Отозвать «${award.achievement.title.ru}» у ${who}?`)) return;
  await api.delete(`/achievements/awards/${award.id}`);
  await load();
}

function rarityBadge(a: Achievement): string {
  return RARITY_BADGE[a.rarity];
}

function rarityLabel(a: Achievement): string {
  return RARITIES.find((r) => r.value === a.rarity)?.label ?? a.rarity;
}

function personLabel(p: { firstName: string | null; username: string | null } | null): string {
  if (!p) return '—';
  return p.firstName ?? (p.username ? `@${p.username}` : '—');
}

onMounted(load);
</script>

<template>
  <div class="flex flex-col gap-8">
    <div>
      <div class="mb-4 flex items-center justify-between">
        <div>
          <h1 class="text-xl font-semibold">Достижения</h1>
          <p class="text-sm text-gray-500">Каталог наград, которые партнёры вручают путешественникам на сайте.</p>
        </div>
        <UButton color="primary" icon="i-heroicons-plus" @click="openCreate">Добавить</UButton>
      </div>
      <p v-if="pageError" class="mb-3 text-sm text-red-600">{{ pageError }}</p>

      <div class="overflow-hidden rounded-lg border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50 text-gray-600">
            <tr>
              <th class="px-4 py-2" />
              <th class="px-4 py-2">Название</th>
              <th class="px-4 py-2">Редкость</th>
              <th class="px-4 py-2">Порядок</th>
              <th class="px-4 py-2">Статус</th>
              <th class="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in achievements" :key="a.id" class="border-t" :class="{ 'opacity-50': !a.isActive }">
              <td class="px-4 py-2"><UIcon :name="a.icon" class="h-6 w-6" /></td>
              <td class="px-4 py-2">
                <p class="font-medium">{{ a.title.ru }}</p>
                <p v-if="a.description.ru" class="text-xs text-gray-500">{{ a.description.ru }}</p>
              </td>
              <td class="px-4 py-2">
                <span class="rounded px-2 py-0.5 text-xs font-medium" :class="rarityBadge(a)">
                  {{ rarityLabel(a) }}
                </span>
              </td>
              <td class="px-4 py-2 text-gray-500">{{ a.sortOrder }}</td>
              <td class="px-4 py-2 text-gray-500">{{ a.isActive ? 'Активно' : 'Скрыто' }}</td>
              <td class="whitespace-nowrap px-4 py-2 text-right">
                <UButton size="xs" variant="ghost" icon="i-heroicons-pencil-square" @click="openEdit(a)" />
                <UButton size="xs" variant="ghost" color="red" icon="i-heroicons-trash" @click="remove(a)" />
              </td>
            </tr>
            <tr v-if="!loading && achievements.length === 0">
              <td colspan="6" class="px-4 py-6 text-center text-gray-400">Достижений пока нет</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div>
      <h2 class="mb-3 text-lg font-semibold">Последние вручения</h2>
      <div class="overflow-hidden rounded-lg border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50 text-gray-600">
            <tr>
              <th class="px-4 py-2">Кому</th>
              <th class="px-4 py-2">Достижение</th>
              <th class="px-4 py-2">Партнёр</th>
              <th class="px-4 py-2">Когда</th>
              <th class="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="award in awards" :key="award.id" class="border-t">
              <td class="px-4 py-2">{{ personLabel(award.user) }}</td>
              <td class="px-4 py-2">
                <span class="inline-flex items-center gap-2">
                  <UIcon :name="award.achievement.icon" class="h-4 w-4" />{{ award.achievement.title.ru }}
                </span>
                <p v-if="award.note" class="text-xs italic text-gray-500">«{{ award.note }}»</p>
              </td>
              <td class="px-4 py-2 text-gray-500">{{ personLabel(award.awardedBy) }}</td>
              <td class="px-4 py-2 text-gray-500">{{ new Date(award.awardedAt).toLocaleString('ru-RU') }}</td>
              <td class="px-4 py-2 text-right">
                <UButton size="xs" variant="ghost" color="red" @click="revoke(award)">Отозвать</UButton>
              </td>
            </tr>
            <tr v-if="!loading && awards.length === 0">
              <td colspan="5" class="px-4 py-6 text-center text-gray-400">Пока никто ничего не вручал</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <UModal v-model="modalOpen" :ui="{ width: 'sm:max-w-2xl' }">
      <form class="flex flex-col gap-4 p-6" @submit.prevent="save">
        <h2 class="text-lg font-semibold">{{ form.id ? 'Редактировать достижение' : 'Новое достижение' }}</h2>

        <div class="grid gap-3 sm:grid-cols-2">
          <UFormGroup v-for="loc in LOCALES" :key="`t-${loc.key}`" :label="`Название · ${loc.label}`">
            <UInput v-model="form.title[loc.key]" :maxlength="80" />
          </UFormGroup>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <UFormGroup v-for="loc in LOCALES" :key="`d-${loc.key}`" :label="`Описание · ${loc.label.replace(' (обязательно)', '')}`">
            <UTextarea v-model="form.description[loc.key]" :rows="2" :maxlength="300" />
          </UFormGroup>
        </div>

        <UFormGroup label="Иконка" hint="Имя иконки Iconify, например i-mdi-trophy">
          <div class="flex items-center gap-3">
            <UIcon :name="form.icon || 'i-mdi-help'" class="h-8 w-8 shrink-0" />
            <UInput v-model="form.icon" class="flex-1" />
          </div>
          <div class="mt-2 flex flex-wrap gap-1">
            <button
              v-for="icon in ICON_IDEAS"
              :key="icon"
              type="button"
              class="rounded border p-1.5 hover:bg-gray-50"
              :class="{ 'border-primary-500 bg-primary-50': form.icon === icon }"
              :title="icon"
              @click="form.icon = icon"
            >
              <UIcon :name="icon" class="h-5 w-5" />
            </button>
          </div>
        </UFormGroup>

        <div class="grid gap-3 sm:grid-cols-3">
          <UFormGroup label="Редкость">
            <USelect v-model="form.rarity" :options="RARITIES" option-attribute="label" value-attribute="value" />
          </UFormGroup>
          <UFormGroup label="Порядок">
            <UInput v-model.number="form.sortOrder" type="number" min="0" />
          </UFormGroup>
          <UFormGroup label="Активно">
            <UToggle v-model="form.isActive" />
          </UFormGroup>
        </div>

        <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="modalOpen = false">Отмена</UButton>
          <UButton type="submit" color="primary" :loading="saving">Сохранить</UButton>
        </div>
      </form>
    </UModal>
  </div>
</template>
