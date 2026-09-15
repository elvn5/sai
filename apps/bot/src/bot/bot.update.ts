import { SupportService } from '@/support/support.service';
import { UsersService } from '@/users/users.service';
import { Injectable, Logger, type OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Action, Ctx, Hears, InjectBot, On, Start, Update } from 'nestjs-telegraf';
import type { Context, Telegraf } from 'telegraf';
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
export class BotUpdate implements OnModuleInit {
  private readonly logger = new Logger(BotUpdate.name);
  private readonly miniAppUrl?: string;

  constructor(
    @InjectBot() private readonly bot: Telegraf<Context>,
    private readonly configService: ConfigService,
    private readonly usersService: UsersService,
    private readonly supportService: SupportService,
    private readonly editSessionService: EditSessionService,
  ) {
    this.miniAppUrl = this.configService.get<string>('MINI_APP_URL');
  }

  async onModuleInit(): Promise<void> {
    if (!this.miniAppUrl) return;

    try {
      await this.bot.telegram.setChatMenuButton({
        menuButton: { type: 'web_app', text: 'Ermekten', web_app: { url: this.miniAppUrl } },
      });
    } catch (error) {
      this.logger.warn(`Failed to set the Mini App chat menu button: ${error}`);
    }
  }

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
      `Здравствуйте, ${from.first_name ?? 'гость'}! Я помогу с гайдами и маршрутами по Кыргызстану и отвечу на вопросы. Выберите раздел в меню ниже.`,
      mainMenuKeyboard(this.miniAppUrl),
    );
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
    await ctx.reply('Редактирование отменено.', mainMenuKeyboard(this.miniAppUrl));
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
      await ctx.reply('Имя обновлено.', mainMenuKeyboard(this.miniAppUrl));
    } else {
      await this.usersService.updatePhone(user.id, text);
      await ctx.reply('Телефон обновлен.', mainMenuKeyboard(this.miniAppUrl));
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
