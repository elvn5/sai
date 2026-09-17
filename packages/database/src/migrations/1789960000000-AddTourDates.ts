import type { MigrationInterface, QueryRunner } from 'typeorm';

export class AddTourDates1789960000000 implements MigrationInterface {
  name = 'AddTourDates1789960000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "tours" ADD "startDate" date`);
    await queryRunner.query(`ALTER TABLE "tours" ADD "endDate" date`);
    // Existing tours predate this field — backfill with today's date so the
    // column can be made NOT NULL without losing any rows.
    await queryRunner.query(
      `UPDATE "tours" SET "startDate" = CURRENT_DATE, "endDate" = CURRENT_DATE WHERE "startDate" IS NULL`,
    );
    await queryRunner.query(`ALTER TABLE "tours" ALTER COLUMN "startDate" SET NOT NULL`);
    await queryRunner.query(`ALTER TABLE "tours" ALTER COLUMN "endDate" SET NOT NULL`);
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "tours" DROP COLUMN "endDate"`);
    await queryRunner.query(`ALTER TABLE "tours" DROP COLUMN "startDate"`);
  }
}
