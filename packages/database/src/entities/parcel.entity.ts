import {
  Column,
  CreateDateColumn,
  Entity,
  JoinColumn,
  ManyToOne,
  PrimaryGeneratedColumn,
  UpdateDateColumn,
} from 'typeorm';
import { User } from './user.entity';

export enum ParcelStatus {
  CREATED = 'created',
  IN_TRANSIT = 'in_transit',
  ARRIVED = 'arrived',
  READY_FOR_PICKUP = 'ready_for_pickup',
  ISSUED = 'issued',
  CANCELLED = 'cancelled',
}

@Entity('parcels')
export class Parcel {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @ManyToOne(
    () => User,
    (user) => user.parcels,
    { onDelete: 'CASCADE' },
  )
  @JoinColumn({ name: 'userId' })
  user!: User;

  @Column({ type: 'uuid' })
  userId!: string;

  @Column({ type: 'varchar' })
  trackNumber!: string;

  @Column({ type: 'varchar', nullable: true })
  description!: string | null;

  @Column({ type: 'enum', enum: ParcelStatus, default: ParcelStatus.CREATED })
  status!: ParcelStatus;

  @Column({ type: 'varchar', nullable: true })
  weightKg!: string | null;

  @Column({ type: 'text', nullable: true })
  comment!: string | null;

  @CreateDateColumn()
  createdAt!: Date;

  @UpdateDateColumn()
  updatedAt!: Date;
}
