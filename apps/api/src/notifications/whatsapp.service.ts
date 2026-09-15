import { Injectable, InternalServerErrorException, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

@Injectable()
export class WhatsappService {
  private readonly logger = new Logger(WhatsappService.name);

  constructor(private readonly configService: ConfigService) {}

  async sendVerificationCode(to: string, code: string): Promise<void> {
    const accountSid = this.configService.get<string>('TWILIO_ACCOUNT_SID');
    const authToken = this.configService.get<string>('TWILIO_AUTH_TOKEN');
    const from = this.configService.get<string>('TWILIO_WHATSAPP_FROM');
    if (!accountSid || !authToken || !from) {
      this.logger.error('Twilio WhatsApp credentials are not configured — cannot send code');
      throw new InternalServerErrorException('WhatsApp service is not configured');
    }

    const body = new URLSearchParams({
      From: from.startsWith('whatsapp:') ? from : `whatsapp:${from}`,
      To: to.startsWith('whatsapp:') ? to : `whatsapp:${to}`,
      Body: `Ваш код подтверждения Ermekten: ${code}. Действителен 10 минут.`,
    });

    const response = await fetch(
      `https://api.twilio.com/2010-04-01/Accounts/${accountSid}/Messages.json`,
      {
        method: 'POST',
        headers: {
          Authorization: `Basic ${Buffer.from(`${accountSid}:${authToken}`).toString('base64')}`,
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body,
      },
    );

    if (!response.ok) {
      const responseBody = await response.text().catch(() => '');
      this.logger.error(`Twilio API error (${response.status}): ${responseBody}`);
      throw new InternalServerErrorException('Could not send verification WhatsApp message');
    }
  }
}
