<script setup lang="ts">
import { ApiError, useApiClient } from '~/shared/api/client';
import type {
  AdminOrder,
  OrderStatus,
  PayeeBalance,
  PayoutRecord,
  Person,
} from '~/shared/api/types';

definePageMeta({ layout: 'default' });

const api = useApiClient();
const orders = ref<AdminOrder[]>([]);
const balances = ref<PayeeBalance[]>([]);
const payouts = ref<PayoutRecord[]>([]);
const status = ref<OrderStatus | ''>('');
const loading = ref(false);
const pageError = ref<string | null>(null);

const STATUSES: { value: OrderStatus | ''; label: string }[] = [
  { value: '', label: 'Все' },
  { value: 'paid', label: 'Оплачено' },
  { value: 'pending', label: 'Ждёт оплаты' },
  { value: 'failed', label: 'Не оплачено' },
  { value: 'expired', label: 'Время вышло' },
  { value: 'refunded', label: 'Возврат' },
];
const STATUS_CLASS: Record<OrderStatus, string> = {
  pending: 'bg-amber-100 text-amber-800',
  paid: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
  expired: 'bg-gray-100 text-gray-600',
  refunded: 'bg-sky-100 text-sky-800',
};

const som = (n: number): string => `${n.toLocaleString('ru-RU')} сом`;
const name = (p?: Person | null): string =>
  p ? (p.firstName ?? (p.username ? `@${p.username}` : p.id.slice(0, 8))) : '—';
const statusClass = (o: AdminOrder): string => STATUS_CLASS[o.status];
const statusLabel = (s: OrderStatus): string => STATUSES.find((x) => x.value === s)?.label ?? s;

async function load(): Promise<void> {
  loading.value = true;
  try {
    [orders.value, balances.value, payouts.value] = await Promise.all([
      api.get<AdminOrder[]>(`/orders/admin${status.value ? `?status=${status.value}` : ''}`),
      api.get<PayeeBalance[]>('/payouts/balances'),
      api.get<PayoutRecord[]>('/payouts'),
    ]);
  } finally {
    loading.value = false;
  }
}

async function payOut(b: PayeeBalance): Promise<void> {
  const note = window.prompt(
    `Отметить выплату ${som(b.amount)} партнёру ${name(b.payee)}?\nКомментарий (например, номер перевода):`,
    '',
  );
  if (note === null) return;
  pageError.value = null;
  try {
    await api.post('/payouts', { payeeId: b.payee.id, note: note.trim() || null });
    await load();
  } catch (error) {
    pageError.value = error instanceof ApiError ? error.message : 'Не удалось записать выплату';
  }
}

async function refund(o: AdminOrder): Promise<void> {
  const note = window.prompt(
    `Отметить возврат ${som(o.amount)} клиенту ${o.contactName}?\nДеньги нужно вернуть вручную. Комментарий:`,
    '',
  );
  if (note === null) return;
  pageError.value = null;
  try {
    await api.post(`/orders/${o.id}/refund`, { note: note.trim() || null });
    await load();
  } catch (error) {
    pageError.value =
      error instanceof ApiError && error.status === 409
        ? 'Возврат возможен только для оплаченных заказов до выплаты партнёру'
        : 'Не удалось отметить возврат';
  }
}

watch(status, load);
onMounted(load);
</script>

<template>
  <div class="flex flex-col gap-8">
    <div>
      <h1 class="text-xl font-semibold">Платежи</h1>
      <p class="text-sm text-gray-500">Онлайн-оплаты броней через Bakai Bank. Деньги приходят на счёт Ermekten; партнёрам выплачивается их часть за вычетом комиссии.</p>
      <p v-if="pageError" class="mt-2 text-sm text-red-600">{{ pageError }}</p>
    </div>

    <div>
      <h2 class="mb-3 text-lg font-semibold">К выплате партнёрам</h2>
      <div class="overflow-hidden rounded-lg border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50 text-gray-600">
            <tr><th class="px-4 py-2">Партнёр</th><th class="px-4 py-2">Заказов</th><th class="px-4 py-2">Сумма</th><th class="px-4 py-2" /></tr>
          </thead>
          <tbody>
            <tr v-for="b in balances" :key="b.payee.id" class="border-t">
              <td class="px-4 py-2">{{ name(b.payee) }} <span class="text-xs text-gray-400">{{ b.payee.id }}</span></td>
              <td class="px-4 py-2 text-gray-500">{{ b.orders }}</td>
              <td class="px-4 py-2 font-semibold">{{ som(b.amount) }}</td>
              <td class="px-4 py-2 text-right"><UButton size="xs" color="primary" @click="payOut(b)">Отметить выплату</UButton></td>
            </tr>
            <tr v-if="!loading && balances.length === 0">
              <td colspan="4" class="px-4 py-6 text-center text-gray-400">Никому ничего не должны</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div>
      <div class="mb-3 flex items-center justify-between">
        <h2 class="text-lg font-semibold">Заказы</h2>
        <USelect v-model="status" :options="STATUSES" option-attribute="label" value-attribute="value" size="sm" />
      </div>
      <div class="overflow-x-auto rounded-lg border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50 text-gray-600">
            <tr>
              <th class="px-4 py-2">Что</th><th class="px-4 py-2">Клиент</th><th class="px-4 py-2">Партнёр</th>
              <th class="px-4 py-2">Сумма</th><th class="px-4 py-2">Статус</th><th class="px-4 py-2">Создан</th><th class="px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="o in orders" :key="o.id" class="border-t">
              <td class="px-4 py-2">
                {{ o.itemTitle }}
                <p class="text-xs text-gray-500">{{ o.itemDate ?? '' }} · {{ o.quantity }} чел.</p>
              </td>
              <td class="px-4 py-2">{{ o.contactName }}<p class="text-xs text-gray-500">{{ o.contactPhone }}</p></td>
              <td class="px-4 py-2 text-gray-600">{{ name(o.payee) }}</td>
              <td class="px-4 py-2">
                {{ som(o.amount) }}
                <p v-if="o.fee" class="text-xs text-gray-500">комиссия {{ som(o.fee) }}</p>
              </td>
              <td class="px-4 py-2">
                <span class="rounded px-2 py-0.5 text-xs font-medium" :class="statusClass(o)">{{ statusLabel(o.status) }}</span>
                <p v-if="o.payoutId" class="text-xs text-gray-500">выплачено</p>
                <p v-if="o.refundNote" class="text-xs italic text-gray-500">{{ o.refundNote }}</p>
              </td>
              <td class="px-4 py-2 text-gray-500">{{ new Date(o.createdAt).toLocaleString('ru-RU') }}</td>
              <td class="px-4 py-2 text-right">
                <UButton v-if="o.status === 'paid' && !o.payoutId" size="xs" variant="ghost" color="red" @click="refund(o)">Возврат</UButton>
              </td>
            </tr>
            <tr v-if="!loading && orders.length === 0">
              <td colspan="7" class="px-4 py-6 text-center text-gray-400">Заказов нет</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div>
      <h2 class="mb-3 text-lg font-semibold">История выплат</h2>
      <div class="overflow-hidden rounded-lg border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-gray-50 text-gray-600">
            <tr><th class="px-4 py-2">Когда</th><th class="px-4 py-2">Партнёр</th><th class="px-4 py-2">Заказов</th><th class="px-4 py-2">Сумма</th><th class="px-4 py-2">Комментарий</th></tr>
          </thead>
          <tbody>
            <tr v-for="p in payouts" :key="p.id" class="border-t">
              <td class="px-4 py-2 text-gray-500">{{ new Date(p.createdAt).toLocaleString('ru-RU') }}</td>
              <td class="px-4 py-2">{{ name(p.payee) }}</td>
              <td class="px-4 py-2 text-gray-500">{{ p.orders }}</td>
              <td class="px-4 py-2 font-semibold">{{ som(p.amount) }}</td>
              <td class="px-4 py-2 text-gray-500">{{ p.note ?? '' }}</td>
            </tr>
            <tr v-if="!loading && payouts.length === 0">
              <td colspan="5" class="px-4 py-6 text-center text-gray-400">Выплат ещё не было</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
