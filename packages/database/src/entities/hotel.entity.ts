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
import { User } from './user.entity';

export enum HotelType {
  HOTEL = 'hotel',
  GUESTHOUSE = 'guesthouse',
  HOSTEL = 'hostel',
  YURT_CAMP = 'yurt_camp',
  RESORT = 'resort',
  APARTMENT = 'apartment',
}

@Entity('hotels')
export class Hotel {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  // Listings are language-independent, so the slug is unique on its own
  // rather than per locale like blog posts and tours.
  @Index({ unique: true })
  @Column({ type: 'varchar' })
  slug!: string;

  @Column({ type: 'varchar' })
  name!: string;

  @Column({ type: 'varchar', default: HotelType.HOTEL })
  type!: HotelType;

  @Column({ type: 'text' })
  description!: string;

  @Index()
  @Column({ type: 'varchar' })
  destination!: string;

  @Column({ type: 'varchar', nullable: true })
  address!: string | null;

  @Column({ type: 'double precision', nullable: true })
  latitude!: number | null;

  @Column({ type: 'double precision', nullable: true })
  longitude!: number | null;

  // Lowest nightly rate, in som.
  @Column({ type: 'int', nullable: true })
  priceFrom!: number | null;

  @Column({ type: 'smallint', nullable: true })
  stars!: number | null;

  @Column({ type: 'simple-array', default: '' })
  amenities!: string[];

  @Column({ type: 'varchar', default: CancellationPolicy.FLEXIBLE })
  cancellationPolicy!: CancellationPolicy;

  @Column({ type: 'varchar', nullable: true })
  checkInTime!: string | null;

  @Column({ type: 'varchar', nullable: true })
  checkOutTime!: string | null;

  @Column({ type: 'varchar', nullable: true })
  coverImageUrl!: string | null;

  @Column({ type: 'simple-json', default: '[]' })
  photos!: string[];

  @Column({ type: 'varchar', nullable: true })
  phone!: string | null;

  @Column({ type: 'varchar', nullable: true })
  whatsapp!: string | null;

  @Column({ type: 'varchar', nullable: true })
  telegram!: string | null;

  @Column({ type: 'varchar', nullable: true })
  website!: string | null;

  // Denormalized from reviews so listings can be filtered and sorted by
  // rating without aggregating on every request.
  @Column({ type: 'real', default: 0 })
  ratingAverage!: number;

  @Column({ type: 'int', default: 0 })
  reviewCount!: number;

  @Column({ type: 'boolean', default: true })
  isPublished!: boolean;

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
