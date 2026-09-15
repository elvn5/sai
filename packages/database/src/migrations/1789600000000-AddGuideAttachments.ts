import { MigrationInterface, QueryRunner } from 'typeorm';

export class AddGuideAttachments1789600000000 implements MigrationInterface {
  name = 'AddGuideAttachments1789600000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(
      `ALTER TABLE "guides" ADD "attachments" text NOT NULL DEFAULT '[]'`,
    );
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "guides" DROP COLUMN "attachments"`);
  }
}
