import { MigrationInterface, QueryRunner } from 'typeorm';

export class AddGalleryRegion1789700000000 implements MigrationInterface {
  name = 'AddGalleryRegion1789700000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(
      `CREATE TYPE "public"."gallery_items_region_enum" AS ENUM('batken', 'jalal_abad', 'issyk_kul', 'naryn', 'osh', 'talas', 'chuy')`,
    );
    await queryRunner.query(
      `ALTER TABLE "gallery_items" ADD "region" "public"."gallery_items_region_enum" NOT NULL DEFAULT 'chuy'`,
    );
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "gallery_items" DROP COLUMN "region"`);
    await queryRunner.query(`DROP TYPE "public"."gallery_items_region_enum"`);
  }
}
