import { AddressesModule } from '@/addresses/addresses.module';
import { BannedItemsModule } from '@/banned-items/banned-items.module';
import { ParcelsModule } from '@/parcels/parcels.module';
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
    ParcelsModule,
    BannedItemsModule,
    AddressesModule,
    SupportModule,
  ],
  providers: [BotUpdate, EditSessionService],
})
export class BotModule {}
