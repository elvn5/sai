import type { MigrationInterface, QueryRunner } from 'typeorm';

export class ReplaceGalleryRegionWithLocation1790000000000 implements MigrationInterface {
  name = 'ReplaceGalleryRegionWithLocation1790000000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "gallery_items" ADD "location" character varying`);
    await queryRunner.query(`ALTER TABLE "gallery_items" DROP COLUMN "region"`);
    await queryRunner.query(`DROP TYPE "public"."gallery_items_region_enum"`);
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(
      `CREATE TYPE "public"."gallery_items_region_enum" AS ENUM('batken', 'jalal_abad', 'issyk_kul', 'naryn', 'osh', 'talas', 'chuy')`,
    );
    await queryRunner.query(
      `ALTER TABLE "gallery_items" ADD "region" "public"."gallery_items_region_enum" NOT NULL DEFAULT 'chuy'`,
    );
    await queryRunner.query(`ALTER TABLE "gallery_items" DROP COLUMN "location"`);
  }
}
