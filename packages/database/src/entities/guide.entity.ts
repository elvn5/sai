import {
  Column,
  CreateDateColumn,
  Entity,
  Index,
  PrimaryGeneratedColumn,
  UpdateDateColumn,
} from 'typeorm';
import { Locale } from './locale.enum';

export enum GuideDifficulty {
  EASY = 'easy',
  MODERATE = 'moderate',
  HARD = 'hard',
}

export enum GuideSeason {
  SUMMER = 'summer',
  WINTER = 'winter',
  ALL_YEAR = 'all_year',
}

@Entity('guides')
@Index(['locale', 'slug'], { unique: true })
export class Guide {
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

  @Column({ type: 'enum', enum: GuideDifficulty, default: GuideDifficulty.EASY })
  difficulty!: GuideDifficulty;

  @Column({ type: 'enum', enum: GuideSeason, default: GuideSeason.ALL_YEAR })
  season!: GuideSeason;

  @Column({ type: 'simple-array', default: '' })
  equipmentChecklist!: string[];

  @Column({ type: 'varchar', nullable: true })
  coverImageUrl!: string | null;

  @Column({ type: 'varchar', nullable: true })
  seoTitle!: string | null;

  @Column({ type: 'varchar', nullable: true })
  seoDescription!: string | null;

  @Column({ type: 'boolean', default: false })
  isPublished!: boolean;

  @Column({ type: 'timestamp', nullable: true })
  publishedAt!: Date | null;

  @CreateDateColumn()
  createdAt!: Date;

  @UpdateDateColumn()
  updatedAt!: Date;
}
