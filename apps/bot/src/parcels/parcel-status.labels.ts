import { ParcelStatus } from '@app/database';

export const PARCEL_STATUS_LABELS: Record<ParcelStatus, string> = {
  [ParcelStatus.CREATED]: 'Оформлена',
  [ParcelStatus.IN_TRANSIT]: 'В пути',
  [ParcelStatus.ARRIVED]: 'Прибыла на склад',
  [ParcelStatus.READY_FOR_PICKUP]: 'Готова к выдаче',
  [ParcelStatus.ISSUED]: 'Выдана',
  [ParcelStatus.CANCELLED]: 'Отменена',
};
