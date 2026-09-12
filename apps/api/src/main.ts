import 'reflect-metadata';
import { mkdirSync } from 'node:fs';
import { ConfigService } from '@nestjs/config';
import { NestFactory } from '@nestjs/core';
import type { NestExpressApplication } from '@nestjs/platform-express';
import cookieParser from 'cookie-parser';
import { Logger } from 'nestjs-pino';
import { AppModule } from './app.module';
import { UPLOADS_DIR } from './uploads/uploads.constants';

async function bootstrap(): Promise<void> {
  mkdirSync(UPLOADS_DIR, { recursive: true });

  const app = await NestFactory.create<NestExpressApplication>(AppModule, { bufferLogs: true });

  app.useLogger(app.get(Logger));
  app.use(cookieParser());
  app.enableCors({
    origin: [
      process.env.ADMIN_ORIGIN ?? 'http://localhost:3000',
      process.env.CLIENT_ORIGIN ?? 'http://localhost:3002',
    ],
    credentials: true,
  });
  app.useStaticAssets(UPLOADS_DIR, { prefix: '/uploads' });

  const configService = app.get(ConfigService);
  const port = configService.get<number>('API_PORT') ?? 3001;

  await app.listen(port);
}

void bootstrap();
