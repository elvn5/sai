import { MigrationInterface, QueryRunner } from 'typeorm';

export class AddGuidesAndUserAvatar1789300000000 implements MigrationInterface {
  name = 'AddGuidesAndUserAvatar1789300000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "users" ADD "avatarUrl" character varying`);

    await queryRunner.query(
      `CREATE TYPE "public"."guides_locale_enum" AS ENUM('ru', 'ky', 'en', 'de')`,
    );
    await queryRunner.query(
      `CREATE TYPE "public"."guides_difficulty_enum" AS ENUM('easy', 'moderate', 'hard')`,
    );
    await queryRunner.query(
      `CREATE TYPE "public"."guides_season_enum" AS ENUM('summer', 'winter', 'all_year')`,
    );
    await queryRunner.query(
      `CREATE TABLE "guides" (
        "id" uuid NOT NULL DEFAULT uuid_generate_v4(),
        "locale" "public"."guides_locale_enum" NOT NULL,
        "slug" character varying NOT NULL,
        "title" character varying NOT NULL,
        "excerpt" text,
        "content" text NOT NULL,
        "difficulty" "public"."guides_difficulty_enum" NOT NULL DEFAULT 'easy',
        "season" "public"."guides_season_enum" NOT NULL DEFAULT 'all_year',
        "equipmentChecklist" text NOT NULL DEFAULT '',
        "coverImageUrl" character varying,
        "seoTitle" character varying,
        "seoDescription" character varying,
        "isPublished" boolean NOT NULL DEFAULT false,
        "publishedAt" TIMESTAMP,
        "createdAt" TIMESTAMP NOT NULL DEFAULT now(),
        "updatedAt" TIMESTAMP NOT NULL DEFAULT now(),
        CONSTRAINT "PK_guides_id" PRIMARY KEY ("id")
      )`,
    );
    await queryRunner.query(
      `CREATE UNIQUE INDEX "IDX_guides_locale_slug" ON "guides" ("locale", "slug")`,
    );
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`DROP INDEX "public"."IDX_guides_locale_slug"`);
    await queryRunner.query(`DROP TABLE "guides"`);
    await queryRunner.query(`DROP TYPE "public"."guides_season_enum"`);
    await queryRunner.query(`DROP TYPE "public"."guides_difficulty_enum"`);
    await queryRunner.query(`DROP TYPE "public"."guides_locale_enum"`);
    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "avatarUrl"`);
  }
}
