import { MigrationInterface, QueryRunner } from 'typeorm';

export class AddGalleryAndGuideOwners1789500000000 implements MigrationInterface {
  name = 'AddGalleryAndGuideOwners1789500000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "gallery_items" ADD "userId" uuid`);
    await queryRunner.query(
      `ALTER TABLE "gallery_items" ADD CONSTRAINT "FK_gallery_items_userId" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE SET NULL ON UPDATE NO ACTION`,
    );

    await queryRunner.query(`ALTER TABLE "guides" ADD "userId" uuid`);
    await queryRunner.query(
      `ALTER TABLE "guides" ADD CONSTRAINT "FK_guides_userId" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE SET NULL ON UPDATE NO ACTION`,
    );
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "guides" DROP CONSTRAINT "FK_guides_userId"`);
    await queryRunner.query(`ALTER TABLE "guides" DROP COLUMN "userId"`);

    await queryRunner.query(`ALTER TABLE "gallery_items" DROP CONSTRAINT "FK_gallery_items_userId"`);
    await queryRunner.query(`ALTER TABLE "gallery_items" DROP COLUMN "userId"`);
  }
}
