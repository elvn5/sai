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

// Kyrgyzstan's 7 regions (oblasts). Every gallery photo belongs to one.
export enum GalleryRegion {
  BATKEN = 'batken',
  JALAL_ABAD = 'jalal_abad',
  ISSYK_KUL = 'issyk_kul',
  NARYN = 'naryn',
  OSH = 'osh',
  TALAS = 'talas',
  CHUY = 'chuy',
}

@Entity('gallery_items')
export class GalleryItem {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Column({ type: 'varchar' })
  imageUrl!: string;

  @Column({ type: 'enum', enum: GalleryRegion, default: GalleryRegion.CHUY })
  region!: GalleryRegion;

  @Column({ type: 'varchar', nullable: true })
  caption!: string | null;

  @Column({ type: 'varchar', nullable: true })
  altText!: string | null;

  @Column({ type: 'int', default: 0 })
  sortOrder!: number;

  @Column({ type: 'boolean', default: true })
  isPublished!: boolean;

  // Null for items created before user submissions existed (curated by admins).
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
