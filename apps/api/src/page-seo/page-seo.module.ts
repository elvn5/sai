import { AuthModule } from '@/auth/auth.module';
import { PageSeo } from '@app/database';
import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { PageSeoController } from './page-seo.controller';
import { PageSeoService } from './page-seo.service';

@Module({
  imports: [TypeOrmModule.forFeature([PageSeo]), AuthModule],
  controllers: [PageSeoController],
  providers: [PageSeoService],
})
export class PageSeoModule {}
