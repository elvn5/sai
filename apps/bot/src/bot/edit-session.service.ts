import { Injectable } from '@nestjs/common';

export type ProfileEditField = 'name' | 'phone';

@Injectable()
export class EditSessionService {
  private readonly pendingEdits = new Map<number, ProfileEditField>();

  start(chatId: number, field: ProfileEditField): void {
    this.pendingEdits.set(chatId, field);
  }

  consume(chatId: number): ProfileEditField | undefined {
    const field = this.pendingEdits.get(chatId);
    this.pendingEdits.delete(chatId);
    return field;
  }

  cancel(chatId: number): void {
    this.pendingEdits.delete(chatId);
  }
}
