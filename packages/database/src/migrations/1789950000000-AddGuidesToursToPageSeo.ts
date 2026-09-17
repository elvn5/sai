import type { MigrationInterface, QueryRunner } from 'typeorm';

export class AddGuidesToursToPageSeo1789950000000 implements MigrationInterface {
  name = 'AddGuidesToursToPageSeo1789950000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(
      `ALTER TYPE "public"."page_seo_page_enum" ADD VALUE 'guides'`,
    );
    await queryRunner.query(
      `ALTER TYPE "public"."page_seo_page_enum" ADD VALUE 'tours'`,
    );
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(
      `ALTER TYPE "public"."page_seo_page_enum" RENAME TO "page_seo_page_enum_old"`,
    );
    await queryRunner.query(
      `CREATE TYPE "public"."page_seo_page_enum" AS ENUM('home', 'blog', 'gallery')`,
    );
    await queryRunner.query(
      `ALTER TABLE "page_seo" ALTER COLUMN "page" TYPE "public"."page_seo_page_enum" USING "page"::text::"public"."page_seo_page_enum"`,
    );
    await queryRunner.query(`DROP TYPE "public"."page_seo_page_enum_old"`);
  }
}
