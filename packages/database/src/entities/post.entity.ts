import {
  Column,
  CreateDateColumn,
  Entity,
  Index,
  PrimaryGeneratedColumn,
  UpdateDateColumn,
} from 'typeorm';

export enum PostLanguage {
  RU = 'ru',
  KY = 'ky',
  EN = 'en',
}

export enum PostStatus {
  DRAFT = 'draft',
  PUBLISHED = 'published',
}

@Entity('posts')
export class Post {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Index({ unique: true })
  @Column({ type: 'varchar' })
  slug!: string;

  @Column({ type: 'varchar' })
  title!: string;

  @Column({ type: 'text', nullable: true })
  excerpt!: string | null;

  /** Санитизированный HTML из TipTap (семантические теги). */
  @Column({ type: 'text' })
  content!: string;

  @Column({ type: 'varchar', nullable: true })
  coverImageUrl!: string | null;

  @Column({ type: 'enum', enum: PostLanguage, default: PostLanguage.RU })
  language!: PostLanguage;

  @Column({ type: 'enum', enum: PostStatus, default: PostStatus.DRAFT })
  status!: PostStatus;

  @Column({ type: 'timestamptz', nullable: true })
  publishedAt!: Date | null;

  @CreateDateColumn()
  createdAt!: Date;

  @UpdateDateColumn()
  updatedAt!: Date;
}
