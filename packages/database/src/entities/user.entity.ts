import {
  Column,
  CreateDateColumn,
  Entity,
  Index,
  OneToMany,
  PrimaryGeneratedColumn,
  UpdateDateColumn,
} from 'typeorm';
import { Parcel } from './parcel.entity';

export enum PartnerStatus {
  NONE = 'none',
  PENDING = 'pending',
  APPROVED = 'approved',
  REJECTED = 'rejected',
}

@Entity('users')
export class User {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Index({ unique: true })
  @Column({ type: 'bigint', nullable: true })
  telegramId!: string | null;

  @Index({ unique: true })
  @Column({ type: 'varchar', nullable: true })
  email!: string | null;

  @Column({ type: 'varchar', nullable: true, select: false })
  passwordHash!: string | null;

  @Column({ type: 'varchar', nullable: true })
  username!: string | null;

  @Column({ type: 'varchar', nullable: true })
  firstName!: string | null;

  @Column({ type: 'varchar', nullable: true })
  lastName!: string | null;

  @Index({ unique: true })
  @Column({ type: 'varchar', nullable: true })
  phone!: string | null;

  @Column({ type: 'varchar', nullable: true })
  avatarUrl!: string | null;

  // True for Telegram-authenticated accounts (verified by Telegram itself)
  // and any account created before this column existed. Email/phone
  // registrations start false and flip to true once the code is confirmed.
  @Column({ type: 'boolean', default: true })
  isVerified!: boolean;

  @Column({ type: 'varchar', nullable: true, select: false })
  verificationCodeHash!: string | null;

  @Column({ type: 'timestamp', nullable: true })
  verificationCodeExpiresAt!: Date | null;

  @Column({ type: 'timestamp', nullable: true })
  verificationCodeSentAt!: Date | null;

  @Column({ type: 'int', default: 0 })
  verificationAttempts!: number;

  @Column({ type: 'enum', enum: PartnerStatus, default: PartnerStatus.NONE })
  partnerStatus!: PartnerStatus;

  @Column({ type: 'timestamp', nullable: true })
  partnerRequestedAt!: Date | null;

  @OneToMany(
    () => Parcel,
    (parcel) => parcel.user,
  )
  parcels!: Parcel[];

  @CreateDateColumn()
  createdAt!: Date;

  @UpdateDateColumn()
  updatedAt!: Date;
}
