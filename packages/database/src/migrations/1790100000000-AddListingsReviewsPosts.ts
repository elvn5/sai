import type { MigrationInterface, QueryRunner } from 'typeorm';

export class AddListingsReviewsPosts1790100000000 implements MigrationInterface {
  name = 'AddListingsReviewsPosts1790100000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(
      `CREATE TABLE "hotels" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "slug" character varying NOT NULL, "name" character varying NOT NULL, "type" character varying NOT NULL DEFAULT 'hotel', "description" text NOT NULL, "destination" character varying NOT NULL, "address" character varying, "latitude" double precision, "longitude" double precision, "priceFrom" integer, "stars" smallint, "amenities" text NOT NULL DEFAULT '', "cancellationPolicy" character varying NOT NULL DEFAULT 'flexible', "checkInTime" character varying, "checkOutTime" character varying, "coverImageUrl" character varying, "photos" text NOT NULL DEFAULT '[]', "phone" character varying, "whatsapp" character varying, "telegram" character varying, "website" character varying, "ratingAverage" real NOT NULL DEFAULT '0', "reviewCount" integer NOT NULL DEFAULT '0', "isPublished" boolean NOT NULL DEFAULT true, "userId" uuid, "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_hotels_id" PRIMARY KEY ("id"))`,
    );
    await queryRunner.query(`CREATE UNIQUE INDEX "IDX_hotels_slug" ON "hotels" ("slug")`);
    await queryRunner.query(`CREATE INDEX "IDX_hotels_destination" ON "hotels" ("destination")`);
    await queryRunner.query(
      `ALTER TABLE "hotels" ADD CONSTRAINT "FK_hotels_userId" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE SET NULL ON UPDATE NO ACTION`,
    );

    await queryRunner.query(
      `CREATE TABLE "local_guides" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "slug" character varying NOT NULL, "name" character varying NOT NULL, "headline" character varying, "bio" text NOT NULL, "avatarUrl" character varying, "coverImageUrl" character varying, "photos" text NOT NULL DEFAULT '[]', "languages" text NOT NULL DEFAULT '', "specializations" text NOT NULL DEFAULT '', "destinations" text NOT NULL DEFAULT '', "pricePerDay" integer, "experienceYears" smallint, "hasVehicle" boolean NOT NULL DEFAULT false, "phone" character varying, "whatsapp" character varying, "telegram" character varying, "ratingAverage" real NOT NULL DEFAULT '0', "reviewCount" integer NOT NULL DEFAULT '0', "isPublished" boolean NOT NULL DEFAULT true, "userId" uuid, "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_local_guides_id" PRIMARY KEY ("id"))`,
    );
    await queryRunner.query(`CREATE UNIQUE INDEX "IDX_local_guides_slug" ON "local_guides" ("slug")`);
    await queryRunner.query(
      `CREATE UNIQUE INDEX "IDX_local_guides_userId" ON "local_guides" ("userId")`,
    );
    await queryRunner.query(
      `ALTER TABLE "local_guides" ADD CONSTRAINT "FK_local_guides_userId" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE SET NULL ON UPDATE NO ACTION`,
    );

    await queryRunner.query(
      `CREATE TABLE "reviews" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "targetType" character varying NOT NULL, "targetId" uuid NOT NULL, "rating" smallint NOT NULL, "text" text NOT NULL, "photos" text NOT NULL DEFAULT '[]', "userId" uuid NOT NULL, "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_reviews_id" PRIMARY KEY ("id"))`,
    );
    await queryRunner.query(`CREATE INDEX "IDX_reviews_targetId" ON "reviews" ("targetId")`);
    await queryRunner.query(
      `CREATE UNIQUE INDEX "IDX_reviews_target_user" ON "reviews" ("targetType", "targetId", "userId")`,
    );
    await queryRunner.query(
      `ALTER TABLE "reviews" ADD CONSTRAINT "FK_reviews_userId" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE NO ACTION`,
    );

    await queryRunner.query(
      `CREATE TABLE "listing_posts" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "targetType" character varying NOT NULL, "targetId" uuid NOT NULL, "text" text NOT NULL, "photos" text NOT NULL DEFAULT '[]', "userId" uuid NOT NULL, "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_listing_posts_id" PRIMARY KEY ("id"))`,
    );
    await queryRunner.query(
      `CREATE INDEX "IDX_listing_posts_target_createdAt" ON "listing_posts" ("targetType", "targetId", "createdAt")`,
    );
    await queryRunner.query(
      `ALTER TABLE "listing_posts" ADD CONSTRAINT "FK_listing_posts_userId" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE NO ACTION`,
    );

    await queryRunner.query(`ALTER TABLE "tours" ADD "destination" character varying`);
    await queryRunner.query(`ALTER TABLE "tours" ADD "cancellationPolicy" character varying`);
    await queryRunner.query(`ALTER TABLE "tours" ADD "phone" character varying`);
    await queryRunner.query(`ALTER TABLE "tours" ADD "whatsapp" character varying`);
    await queryRunner.query(`ALTER TABLE "tours" ADD "telegram" character varying`);
    await queryRunner.query(`ALTER TABLE "tours" ADD "ratingAverage" real NOT NULL DEFAULT '0'`);
    await queryRunner.query(`ALTER TABLE "tours" ADD "reviewCount" integer NOT NULL DEFAULT '0'`);
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "tours" DROP COLUMN "reviewCount"`);
    await queryRunner.query(`ALTER TABLE "tours" DROP COLUMN "ratingAverage"`);
    await queryRunner.query(`ALTER TABLE "tours" DROP COLUMN "telegram"`);
    await queryRunner.query(`ALTER TABLE "tours" DROP COLUMN "whatsapp"`);
    await queryRunner.query(`ALTER TABLE "tours" DROP COLUMN "phone"`);
    await queryRunner.query(`ALTER TABLE "tours" DROP COLUMN "cancellationPolicy"`);
    await queryRunner.query(`ALTER TABLE "tours" DROP COLUMN "destination"`);
    await queryRunner.query(`DROP TABLE "listing_posts"`);
    await queryRunner.query(`DROP TABLE "reviews"`);
    await queryRunner.query(`DROP TABLE "local_guides"`);
    await queryRunner.query(`DROP TABLE "hotels"`);
  }
}
