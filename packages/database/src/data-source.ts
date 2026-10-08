import 'reflect-metadata';
import { config } from 'dotenv';
import { DataSource, type DataSourceOptions } from 'typeorm';
import { Address } from './entities/address.entity';
import { BannedItem } from './entities/banned-item.entity';
import { BlogPost } from './entities/blog-post.entity';
import { GalleryItem } from './entities/gallery-item.entity';
import { Guide } from './entities/guide.entity';
import { Hotel } from './entities/hotel.entity';
import { ListingPost } from './entities/listing-post.entity';
import { LocalGuide } from './entities/local-guide.entity';
import { PageSeo } from './entities/page-seo.entity';
import { Parcel } from './entities/parcel.entity';
import { Review } from './entities/review.entity';
import { SupportInfo } from './entities/support-info.entity';
import { Tour } from './entities/tour.entity';
import { User } from './entities/user.entity';

config();

export const entities = [
  User,
  Parcel,
  Address,
  BannedItem,
  SupportInfo,
  BlogPost,
  GalleryItem,
  Guide,
  Tour,
  PageSeo,
  Hotel,
  LocalGuide,
  Review,
  ListingPost,
];

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
