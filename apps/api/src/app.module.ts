import { AddressesModule } from '@/addresses/addresses.module';
import { AuthModule } from '@/auth/auth.module';
import { BannedItemsModule } from '@/banned-items/banned-items.module';
import { validateEnv } from '@/config/env.schema';
import { ParcelsModule } from '@/parcels/parcels.module';
import { SupportModule } from '@/support/support.module';
import { UsersModule } from '@/users/users.module';
import { entities } from '@app/database';
import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { TypeOrmModule } from '@nestjs/typeorm';
import { LoggerModule } from 'nestjs-pino';

@Module({
  imports: [
    ConfigModule.forRoot({ isGlobal: true, validate: validateEnv }),
    LoggerModule.forRoot({
      pinoHttp: {
        transport: process.env.NODE_ENV === 'production' ? undefined : { target: 'pino-pretty' },
        level: process.env.NODE_ENV === 'production' ? 'info' : 'debug',
      },
    }),
    TypeOrmModule.forRootAsync({
      inject: [ConfigService],
      useFactory: (configService: ConfigService) => ({
        type: 'postgres' as const,
        host: configService.get<string>('POSTGRES_HOST'),
        port: configService.get<number>('POSTGRES_PORT'),
        username: configService.get<string>('POSTGRES_USER'),
        password: configService.get<string>('POSTGRES_PASSWORD'),
        database: configService.get<string>('POSTGRES_DB'),
        entities,
        synchronize: false,
        autoLoadEntities: true,
      }),
    }),
    AuthModule,
    BannedItemsModule,
    AddressesModule,
    SupportModule,
    ParcelsModule,
    UsersModule,
  ],
})
export class AppModule {}
