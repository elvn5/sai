import { AddressesService } from '@/addresses/addresses.service';
import { BannedItemsService } from '@/banned-items/banned-items.service';
import { PARCEL_STATUS_LABELS } from '@/parcels/parcel-status.labels';
import { ParcelsService } from '@/parcels/parcels.service';
import { SupportService } from '@/support/support.service';
import { UsersService } from '@/users/users.service';
import { Injectable, Logger } from '@nestjs/common';
import { Action, Ctx, Hears, On, Start, Update } from 'nestjs-telegraf';
import type { Context } from 'telegraf';
import type { Message } from 'telegraf/typings/core/types/typegram';
import { EditSessionService } from './edit-session.service';
import {
  MENU_BUTTONS,
  cancelEditKeyboard,
  mainMenuKeyboard,
  profileEditKeyboard,
} from './keyboards';

@Update()
@Injectable()
export class BotUpdate {
  private readonly logger = new Logger(BotUpdate.name);

  constructor(
    private readonly usersService: UsersService,
    private readonly parcelsService: ParcelsService,
    private readonly bannedItemsService: BannedItemsService,
    private readonly addressesService: AddressesService,
    private readonly supportService: SupportService,
    private readonly editSessionService: EditSessionService,
  ) {}

  @Start()
  async onStart(@Ctx() ctx: Context): Promise<void> {
    const from = ctx.from;
    if (!from) return;

    await this.usersService.findOrCreate({
      telegramId: String(from.id),
      username: from.username,
      firstName: from.first_name,
      lastName: from.last_name,
    });

    await ctx.reply(
      `Здравствуйте, ${from.first_name ?? 'гость'}! Я помогу отследить посылки и отвечу на вопросы. Выберите раздел в меню ниже.`,
      mainMenuKeyboard,
    );
  }

  @Hears(MENU_BUTTONS.PARCELS)
  async onParcels(@Ctx() ctx: Context): Promise<void> {
    const user = await this.getOrRegisterUser(ctx);
    if (!user) return;

    const parcels = await this.parcelsService.findByUserId(user.id);

    if (parcels.length === 0) {
      await ctx.reply('У вас пока нет посылок.');
      return;
    }

    const lines = parcels.map((parcel, index) => {
      const status = PARCEL_STATUS_LABELS[parcel.status];
      const description = parcel.description ? ` — ${parcel.description}` : '';
      return `${index + 1}. Трек-номер: ${parcel.trackNumber}\nСтатус: ${status}${description}`;
    });

    await ctx.reply(`Ваши посылки:\n\n${lines.join('\n\n')}`);
  }

  @Hears(MENU_BUTTONS.BANNED_ITEMS)
  async onBannedItems(@Ctx() ctx: Context): Promise<void> {
    const items = await this.bannedItemsService.findActive();

    if (items.length === 0) {
      await ctx.reply('Список запрещенных товаров пока пуст.');
      return;
    }

    const lines = items.map((item, index) => {
      const description = item.description ? ` — ${item.description}` : '';
      return `${index + 1}. ${item.name}${description}`;
    });

    await ctx.reply(`Запрещенные к пересылке товары:\n\n${lines.join('\n')}`);
  }

  @Hears(MENU_BUTTONS.ADDRESSES)
  async onAddresses(@Ctx() ctx: Context): Promise<void> {
    const addresses = await this.addressesService.findActive();

    if (addresses.length === 0) {
      await ctx.reply('Адреса пока не настроены.');
      return;
    }

    const lines = addresses.map((address) => {
      const comment = address.comment ? `\n${address.comment}` : '';
      return `📍 ${address.title}\n${address.fullAddress}${comment}`;
    });

    await ctx.reply(lines.join('\n\n'));
  }

  @Hears(MENU_BUTTONS.PROFILE)
  async onProfile(@Ctx() ctx: Context): Promise<void> {
    const user = await this.getOrRegisterUser(ctx);
    if (!user) return;

    const lines = [
      `Имя: ${user.firstName ?? 'не указано'}`,
      `Username: ${user.username ? `@${user.username}` : 'не указан'}`,
      `Телефон: ${user.phone ?? 'не указан'}`,
    ];

    await ctx.reply(`Ваш профиль:\n\n${lines.join('\n')}`, profileEditKeyboard);
  }

  @Hears(MENU_BUTTONS.SUPPORT)
  async onSupport(@Ctx() ctx: Context): Promise<void> {
    const support = await this.supportService.get();

    const lines = [support.message];
    if (support.phone) lines.push(`Телефон: ${support.phone}`);
    if (support.telegramContact) lines.push(`Telegram: ${support.telegramContact}`);
    if (support.email) lines.push(`Email: ${support.email}`);
    if (support.workingHours) lines.push(`Часы работы: ${support.workingHours}`);

    await ctx.reply(lines.join('\n'));
  }

  @Action('profile_edit_name')
  async onEditName(@Ctx() ctx: Context): Promise<void> {
    if (!ctx.chat) return;
    this.editSessionService.start(ctx.chat.id, 'name');
    await ctx.answerCbQuery();
    await ctx.reply('Введите новое имя:', cancelEditKeyboard);
  }

  @Action('profile_edit_phone')
  async onEditPhone(@Ctx() ctx: Context): Promise<void> {
    if (!ctx.chat) return;
    this.editSessionService.start(ctx.chat.id, 'phone');
    await ctx.answerCbQuery();
    await ctx.reply('Введите новый номер телефона:', cancelEditKeyboard);
  }

  @Action('profile_edit_cancel')
  async onEditCancel(@Ctx() ctx: Context): Promise<void> {
    if (!ctx.chat) return;
    this.editSessionService.cancel(ctx.chat.id);
    await ctx.answerCbQuery('Отменено');
    await ctx.reply('Редактирование отменено.', mainMenuKeyboard);
  }

  @On('text')
  async onText(@Ctx() ctx: Context): Promise<void> {
    if (!ctx.chat) return;

    const pendingField = this.editSessionService.consume(ctx.chat.id);
    if (!pendingField) return;

    const user = await this.getOrRegisterUser(ctx);
    if (!user) return;

    const text = (ctx.message as Message.TextMessage | undefined)?.text?.trim();
    if (!text) {
      await ctx.reply('Пустое значение не сохранено.');
      return;
    }

    if (pendingField === 'name') {
      await this.usersService.updateFirstName(user.id, text);
      await ctx.reply('Имя обновлено.', mainMenuKeyboard);
    } else {
      await this.usersService.updatePhone(user.id, text);
      await ctx.reply('Телефон обновлен.', mainMenuKeyboard);
    }
  }

  private async getOrRegisterUser(ctx: Context) {
    const from = ctx.from;
    if (!from) {
      this.logger.warn('Received update without ctx.from');
      return null;
    }

    return this.usersService.findOrCreate({
      telegramId: String(from.id),
      username: from.username,
      firstName: from.first_name,
      lastName: from.last_name,
    });
  }
}
