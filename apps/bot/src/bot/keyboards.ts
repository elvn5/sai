import { Markup } from 'telegraf';

export const MENU_BUTTONS = {
  PARCELS: '📦 Мои посылки',
  BANNED_ITEMS: '🚫 Запрещенные товары',
  ADDRESSES: '📍 Адреса',
  PROFILE: '👤 Профиль',
  SUPPORT: '🆘 Поддержка',
} as const;

export const mainMenuKeyboard = Markup.keyboard([
  [MENU_BUTTONS.PARCELS, MENU_BUTTONS.BANNED_ITEMS],
  [MENU_BUTTONS.ADDRESSES, MENU_BUTTONS.PROFILE],
  [MENU_BUTTONS.SUPPORT],
]).resize();

export const profileEditKeyboard = Markup.inlineKeyboard([
  [Markup.button.callback('✏️ Изменить имя', 'profile_edit_name')],
  [Markup.button.callback('✏️ Изменить телефон', 'profile_edit_phone')],
]);

export const cancelEditKeyboard = Markup.inlineKeyboard([
  [Markup.button.callback('Отмена', 'profile_edit_cancel')],
]);
