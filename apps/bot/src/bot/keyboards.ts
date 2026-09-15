import { Markup } from 'telegraf';

export const MENU_BUTTONS = {
  PARCELS: '📦 Мои посылки',
  BANNED_ITEMS: '🚫 Запрещенные товары',
  ADDRESSES: '📍 Адреса',
  PROFILE: '👤 Профиль',
  SUPPORT: '🆘 Поддержка',
  MINI_APP: '🏔 Открыть Ermekten',
} as const;

export function mainMenuKeyboard(miniAppUrl?: string) {
  const textRows = [
    [MENU_BUTTONS.PARCELS, MENU_BUTTONS.BANNED_ITEMS],
    [MENU_BUTTONS.ADDRESSES, MENU_BUTTONS.PROFILE],
    [MENU_BUTTONS.SUPPORT],
  ];

  if (!miniAppUrl) {
    return Markup.keyboard(textRows).resize();
  }

  return Markup.keyboard([
    ...textRows,
    [Markup.button.webApp(MENU_BUTTONS.MINI_APP, miniAppUrl)],
  ]).resize();
}

export const profileEditKeyboard = Markup.inlineKeyboard([
  [Markup.button.callback('✏️ Изменить имя', 'profile_edit_name')],
  [Markup.button.callback('✏️ Изменить телефон', 'profile_edit_phone')],
]);

export const cancelEditKeyboard = Markup.inlineKeyboard([
  [Markup.button.callback('Отмена', 'profile_edit_cancel')],
]);
