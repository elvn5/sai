import { Injectable, InternalServerErrorException, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

@Injectable()
export class EmailService {
  private readonly logger = new Logger(EmailService.name);

  constructor(private readonly configService: ConfigService) {}

  async sendVerificationCode(to: string, code: string): Promise<void> {
    const apiKey = this.configService.get<string>('RESEND_API_KEY');
    if (!apiKey) {
      this.logger.error('RESEND_API_KEY is not configured — cannot send verification email');
      throw new InternalServerErrorException('Email service is not configured');
    }

    const from = this.configService.get<string>('EMAIL_FROM');
    const response = await fetch('https://api.resend.com/emails', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        from,
        to: [to],
        subject: 'Код подтверждения Ermekten',
        html: `<p>Ваш код подтверждения: <strong>${code}</strong></p><p>Код действителен 10 минут.</p>`,
      }),
    });

    if (!response.ok) {
      const body = await response.text().catch(() => '');
      this.logger.error(`Resend API error (${response.status}): ${body}`);
      throw new InternalServerErrorException('Could not send verification email');
    }
  }
}
