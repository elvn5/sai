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

// News/photo updates a listing's owner posts on their hotel, guide or tour
// page. Same polymorphic target as Review.
@Entity('listing_posts')
@Index(['targetType', 'targetId', 'createdAt'])
export class ListingPost {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Column({ type: 'varchar' })
  targetType!: ReviewTargetType;

  @Column({ type: 'uuid' })
  targetId!: string;

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
