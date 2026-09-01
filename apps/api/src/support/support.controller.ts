import { JwtAuthGuard } from '@/auth/jwt-auth.guard';
import { ZodValidationPipe } from '@/common/zod-validation.pipe';
import { Body, Controller, Get, Put, UseGuards } from '@nestjs/common';
import { type UpdateSupportInfoDto, updateSupportInfoSchema } from './support.dto';
import { SupportService } from './support.service';

@Controller('support')
@UseGuards(JwtAuthGuard)
export class SupportController {
  constructor(private readonly service: SupportService) {}

  @Get()
  get() {
    return this.service.get();
  }

  @Put()
  update(@Body(new ZodValidationPipe(updateSupportInfoSchema)) dto: UpdateSupportInfoDto) {
    return this.service.update(dto);
  }
}
