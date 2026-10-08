export function whatsappLink(phone: string, message?: string): string {
  const digits = phone.replace(/\D/g, '');
  const text = message ? `?text=${encodeURIComponent(message)}` : '';
  return `https://wa.me/${digits}${text}`;
}

export function telegramLink(username: string): string {
  return `https://t.me/${username.replace(/^@/, '')}`;
}

export function phoneLink(phone: string): string {
  return `tel:${phone.replace(/[^\d+]/g, '')}`;
}

export function mapsLink(
  latitude: number | null,
  longitude: number | null,
  address: string | null,
): string | null {
  if (latitude !== null && longitude !== null) {
    return `https://www.google.com/maps/search/?api=1&query=${latitude},${longitude}`;
  }
  if (address) {
    return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address)}`;
  }
  return null;
}
