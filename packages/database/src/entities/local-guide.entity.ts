import {
  Column,
  CreateDateColumn,
  Entity,
  Index,
  JoinColumn,
  ManyToOne,
  PrimaryGeneratedColumn,
  UpdateDateColumn,
} from 'typeorm';
import { User } from './user.entity';

// A person offering guide services — not to be confused with Guide, which
// is a written route/how-to article.
@Entity('local_guides')
export class LocalGuide {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Index({ unique: true })
  @Column({ type: 'varchar' })
  slug!: string;

  @Column({ type: 'varchar' })
  name!: string;

  @Column({ type: 'varchar', nullable: true })
  headline!: string | null;

  @Column({ type: 'text' })
  bio!: string;

  @Column({ type: 'varchar', nullable: true })
  avatarUrl!: string | null;

  @Column({ type: 'varchar', nullable: true })
  coverImageUrl!: string | null;

  @Column({ type: 'simple-json', default: '[]' })
  photos!: string[];

  @Column({ type: 'simple-array', default: '' })
  languages!: string[];

  @Column({ type: 'simple-array', default: '' })
  specializations!: string[];

  @Column({ type: 'simple-array', default: '' })
  destinations!: string[];

  // Daily rate, in som.
  @Column({ type: 'int', nullable: true })
  pricePerDay!: number | null;

  @Column({ type: 'smallint', nullable: true })
  experienceYears!: number | null;

  // Guide-driver: comes with their own 4WD.
  @Column({ type: 'boolean', default: false })
  hasVehicle!: boolean;

  @Column({ type: 'varchar', nullable: true })
  phone!: string | null;

  @Column({ type: 'varchar', nullable: true })
  whatsapp!: string | null;

  @Column({ type: 'varchar', nullable: true })
  telegram!: string | null;

  @Column({ type: 'real', default: 0 })
  ratingAverage!: number;

  @Column({ type: 'int', default: 0 })
  reviewCount!: number;

  @Column({ type: 'boolean', default: true })
  isPublished!: boolean;

  @ManyToOne(() => User, { nullable: true, onDelete: 'SET NULL' })
  @JoinColumn({ name: 'userId' })
  user!: User | null;

  // One guide profile per user.
  @Index({ unique: true })
  @Column({ type: 'uuid', nullable: true })
  userId!: string | null;

  @CreateDateColumn()
  createdAt!: Date;

  @UpdateDateColumn()
  updatedAt!: Date;
}
