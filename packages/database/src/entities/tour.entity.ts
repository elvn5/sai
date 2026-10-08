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
import { CancellationPolicy } from './listing.enums';
import { Locale } from './locale.enum';
import { User } from './user.entity';

@Entity('tours')
@Index(['locale', 'slug'], { unique: true })
export class Tour {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Column({ type: 'enum', enum: Locale })
  locale!: Locale;

  @Column({ type: 'varchar' })
  slug!: string;

  @Column({ type: 'varchar' })
  title!: string;

  @Column({ type: 'text', nullable: true })
  excerpt!: string | null;

  @Column({ type: 'text' })
  content!: string;

  @Column({ type: 'int', nullable: true })
  price!: number | null;

  @Column({ type: 'int', nullable: true })
  durationDays!: number | null;

  @Column({ type: 'varchar', nullable: true })
  location!: string | null;

  @Column({ type: 'date' })
  startDate!: string;

  @Column({ type: 'date' })
  endDate!: string;

  @Column({ type: 'varchar', nullable: true })
  destination!: string | null;

  @Column({ type: 'varchar', nullable: true })
  cancellationPolicy!: CancellationPolicy | null;

  @Column({ type: 'varchar', nullable: true })
  coverImageUrl!: string | null;

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

  @Column({ type: 'boolean', default: false })
  isPublished!: boolean;

  @Column({ type: 'timestamp', nullable: true })
  publishedAt!: Date | null;

  // Tours are only ever created by approved partners, but kept nullable to
  // match the SET NULL pattern used by other user-generated content — a
  // deleted partner's past tours stay up instead of cascading away.
  @ManyToOne(() => User, { nullable: true, onDelete: 'SET NULL' })
  @JoinColumn({ name: 'userId' })
  user!: User | null;

  @Column({ type: 'uuid', nullable: true })
  userId!: string | null;

  @CreateDateColumn()
  createdAt!: Date;

  @UpdateDateColumn()
  updatedAt!: Date;
}
