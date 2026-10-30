import type { TransactionStatus } from '@/core/api/types';

export const TRANSACTION_STATUSES: { value: TransactionStatus; label: string; tone: string }[] = [
  { value: 'CREATED', label: 'Ждёт оплаты', tone: 'bg-amber-100 text-amber-800' },
  { value: 'PENDING', label: 'В обработке', tone: 'bg-amber-100 text-amber-800' },
  { value: 'COMMITTED', label: 'Оплачен', tone: 'bg-green-100 text-green-800' },
  { value: 'FAILED', label: 'Отклонён', tone: 'bg-red-100 text-red-800' },
  { value: 'EXPIRED', label: 'Истёк', tone: 'bg-gray-100 text-gray-600' },
  { value: 'CANCELLED', label: 'Отменён', tone: 'bg-gray-100 text-gray-600' },
];

export function transactionStatusLabel(status: string): string {
  return TRANSACTION_STATUSES.find((s) => s.value === status)?.label ?? status;
}

export function transactionStatusTone(status: string): string {
  return TRANSACTION_STATUSES.find((s) => s.value === status)?.tone ?? 'bg-gray-100 text-gray-600';
}

// GoPay's amount ("24000.00") in som.
export function gatewayAmount(amount: string): number {
  return Number(amount);
}

const GATEWAYS: Record<string, string> = {
  gopay: 'GoPay',
  bakai: 'Bakai Bank',
  fake: 'тестовая страница (без денег)',
  off: 'выключен',
};

export function gatewayName(provider: string): string {
  return GATEWAYS[provider] ?? provider;
}

const ORDER_LABELS: Record<string, string> = {
  pending: 'ждёт оплаты',
  paid: 'оплачен',
  failed: 'не оплачен',
  expired: 'время вышло',
  refunded: 'возврат',
};

export function orderStatusLabel(status: string): string {
  return ORDER_LABELS[status] ?? status;
}
