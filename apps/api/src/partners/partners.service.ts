import { PartnerStatus, User } from '@app/database';
import { ConflictException, Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import type { Repository } from 'typeorm';

@Injectable()
export class PartnersService {
  constructor(
    @InjectRepository(User)
    private readonly users: Repository<User>,
  ) {}

  async apply(userId: string): Promise<User> {
    const user = await this.findOrThrow(userId);
    if (
      user.partnerStatus === PartnerStatus.PENDING ||
      user.partnerStatus === PartnerStatus.APPROVED
    ) {
      throw new ConflictException('Partner request already submitted');
    }

    user.partnerStatus = PartnerStatus.PENDING;
    user.partnerRequestedAt = new Date();
    return this.users.save(user);
  }

  findPending(): Promise<User[]> {
    return this.users.find({
      where: { partnerStatus: PartnerStatus.PENDING },
      order: { partnerRequestedAt: 'ASC' },
    });
  }

  async approve(userId: string): Promise<User> {
    const user = await this.findOrThrow(userId);
    user.partnerStatus = PartnerStatus.APPROVED;
    return this.users.save(user);
  }

  async reject(userId: string): Promise<User> {
    const user = await this.findOrThrow(userId);
    user.partnerStatus = PartnerStatus.REJECTED;
    return this.users.save(user);
  }

  private async findOrThrow(userId: string): Promise<User> {
    const user = await this.users.findOne({ where: { id: userId } });
    if (!user) {
      throw new NotFoundException('User not found');
    }
    return user;
  }
}
