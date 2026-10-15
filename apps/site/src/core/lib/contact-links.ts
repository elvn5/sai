export function whatsappLink(phone: string, message?: string): string {
  const digits = phone.replace(/\D/g, '');
  return `https://wa.me/${digits}${message ? `?text=${encodeURIComponent(message)}` : ''}`;
}

export function telegramLink(username: string): string {
  return `https://t.me/${username.replace(/^@/, '')}`;
}

export function phoneLink(phone: string): string {
  return `tel:${phone.replace(/[^\d+]/g, '')}`;
}

// Google Maps search for the point, or the address when there's no point.
export function mapsLink(
  latitude: number | null,
  longitude: number | null,
  address: string | null,
): string | null {
  if (latitude !== null && longitude !== null) {
    return `https://www.google.com/maps/search/?api=1&query=${latitude},${longitude}`;
  }
  return address
    ? `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address)}`
    : null;
}
