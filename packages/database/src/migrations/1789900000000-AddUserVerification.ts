import { MigrationInterface, QueryRunner } from 'typeorm';

export class AddUserVerification1789900000000 implements MigrationInterface {
  name = 'AddUserVerification1789900000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "users" ADD "isVerified" boolean NOT NULL DEFAULT true`);
    await queryRunner.query(`ALTER TABLE "users" ADD "verificationCodeHash" character varying`);
    await queryRunner.query(`ALTER TABLE "users" ADD "verificationCodeExpiresAt" TIMESTAMP`);
    await queryRunner.query(`ALTER TABLE "users" ADD "verificationCodeSentAt" TIMESTAMP`);
    await queryRunner.query(
      `ALTER TABLE "users" ADD "verificationAttempts" integer NOT NULL DEFAULT 0`,
    );
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "verificationAttempts"`);
    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "verificationCodeSentAt"`);
    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "verificationCodeExpiresAt"`);
    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "verificationCodeHash"`);
    await queryRunner.query(`ALTER TABLE "users" DROP COLUMN "isVerified"`);
  }
}
