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
import { ReviewTargetType } from './listing.enums';
import { User } from './user.entity';

// Polymorphic: targetId points at a Hotel, LocalGuide or Tour depending on
// targetType. Owners of those listings clean up reviews when deleting.
@Entity('reviews')
@Index(['targetType', 'targetId', 'userId'], { unique: true })
export class Review {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Column({ type: 'varchar' })
  targetType!: ReviewTargetType;

  @Index()
  @Column({ type: 'uuid' })
  targetId!: string;

  @Column({ type: 'smallint' })
  rating!: number;

  @Column({ type: 'text' })
  text!: string;

  @Column({ type: 'simple-json', default: '[]' })
  photos!: string[];

  @ManyToOne(() => User, { onDelete: 'CASCADE' })
  @JoinColumn({ name: 'userId' })
  user!: User;

  @Column({ type: 'uuid' })
  userId!: string;

  @CreateDateColumn()
  createdAt!: Date;

  @UpdateDateColumn()
  updatedAt!: Date;
}
