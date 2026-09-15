import { MigrationInterface, QueryRunner } from 'typeorm';

export class AddPartnersAndTours1789800000000 implements MigrationInterface {
  name = 'AddPartnersAndTours1789800000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(
      `CREATE TYPE "public"."users_partnerstatus_enum" AS ENUM('none', 'pending', 'approved', 'rejected')`,
    );
    await queryRunner.query(
      `ALTER TABLE "users" ADD "partnerStatus" "public"."users_partnerstatus_enum" NOT NULL DEFAULT 'none'`,
    );
    await queryRunner.query(`ALTER TABLE "users" ADD "partnerRequestedAt" TIMESTAMP`);

    await queryRunner.query(`CREATE TYPE "public"."tours_locale_enum" AS ENUM('ru', 'ky', 'en', 'de')`);
    await queryRunner.query(`
      CREATE TABLE "tours" (
        "id" uuid NOT NULL DEFAULT uuid_generate_v4(),
        "locale" "public"."tours_locale_enum" NOT NULL,
        "slug" character varying NOT NULL,
        "title" character varying NOT NULL,
        "excerpt" text,
        "content" text NOT NULL,
        "price" integer,
        "durationDays" integer,
        "location" character varying,
        "coverImageUrl" character varying,
        "isPublished" boolean NOT NULL DEFAULT false,
        "publishedAt" TIMESTAMP,
        "userId" uuid,
        "createdAt" TIMESTAMP NOT NULL DEFAULT now(),
        "updatedAt" TIMESTAMP NOT NULL DEFAULT now(),
        CONSTRAINT "PK_tours_id" PRIMARY KEY ("id")
      )
    `);
    await queryRunner.query(`CREATE UNIQUE INDEX "IDX_tours_locale_slug" ON "tours" ("locale", "slug")`);
    await queryRunner.query(
      `ALTER TABLE "tours" ADD CONSTRAINT "FK_tours_userId" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE SET NULL ON UPDATE NO ACTION`,
    );
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "tours" DROP CONSTRAINT "FK_tours_userId"`);
    await queryRunner.query(`DROP INDEX "public"."IDX_tours_locale_slug"`);
    await queryRunner.query(`DROP TABLE "tours"`);
    await queryRunner.query(`DROP TYPE "public"."tours_locale_enum"`);

    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "partnerRequestedAt"`);
    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "partnerStatus"`);
    await queryRunner.query(`DROP TYPE "public"."users_partnerstatus_enum"`);
  }
}
