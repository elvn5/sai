import 'reflect-metadata';
import { config } from 'dotenv';
import { DataSource, type DataSourceOptions } from 'typeorm';
import { Address } from './entities/address.entity';
import { BannedItem } from './entities/banned-item.entity';
import { Parcel } from './entities/parcel.entity';
import { SupportInfo } from './entities/support-info.entity';
import { User } from './entities/user.entity';

config();

export const entities = [User, Parcel, Address, BannedItem, SupportInfo];

export function buildDataSourceOptions(): DataSourceOptions {
  return {
    type: 'postgres',
    host: process.env.POSTGRES_HOST ?? 'localhost',
    port: Number(process.env.POSTGRES_PORT ?? 5432),
    username: process.env.POSTGRES_USER ?? 'parcel_bot',
    password: process.env.POSTGRES_PASSWORD ?? 'change_me',
    database: process.env.POSTGRES_DB ?? 'parcel_bot',
    entities,
    migrations: [`${__dirname}/migrations/*.{ts,js}`],
    synchronize: false,
    logging: false,
  };
}

const AppDataSource = new DataSource(buildDataSourceOptions());

export default AppDataSource;
