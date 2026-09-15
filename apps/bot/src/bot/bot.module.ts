import { SupportModule } from '@/support/support.module';
import { UsersModule } from '@/users/users.module';
import { Module } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { TelegrafModule } from 'nestjs-telegraf';
import { BotUpdate } from './bot.update';
import { EditSessionService } from './edit-session.service';

@Module({
  imports: [
    TelegrafModule.forRootAsync({
      inject: [ConfigService],
      useFactory: (configService: ConfigService) => ({
        token: configService.get<string>('BOT_TOKEN') ?? '',
      }),
    }),
    UsersModule,
    SupportModule,
  ],
  providers: [BotUpdate, EditSessionService],
})
export class BotModule {}
