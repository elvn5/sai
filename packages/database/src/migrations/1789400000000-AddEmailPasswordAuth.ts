import { MigrationInterface, QueryRunner } from 'typeorm';

export class AddEmailPasswordAuth1789400000000 implements MigrationInterface {
  name = 'AddEmailPasswordAuth1789400000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "users" ALTER COLUMN "telegramId" DROP NOT NULL`);
    await queryRunner.query(`ALTER TABLE "users" ADD "email" character varying`);
    await queryRunner.query(`ALTER TABLE "users" ADD "passwordHash" character varying`);
    await queryRunner.query(`CREATE UNIQUE INDEX "IDX_users_email" ON "users" ("email")`);
    await queryRunner.query(`CREATE UNIQUE INDEX "IDX_users_phone" ON "users" ("phone")`);
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`DROP INDEX "public"."IDX_users_phone"`);
    await queryRunner.query(`DROP INDEX "public"."IDX_users_email"`);
    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "passwordHash"`);
    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "email"`);
    await queryRunner.query(`ALTER TABLE "users" ALTER COLUMN "telegramId" SET NOT NULL`);
  }
}
