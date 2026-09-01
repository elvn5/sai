import { MigrationInterface, QueryRunner } from "typeorm";

export class InitSchema1786809560789 implements MigrationInterface {
    name = 'InitSchema1786809560789'

    public async up(queryRunner: QueryRunner): Promise<void> {
        await queryRunner.query(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`);
        await queryRunner.query(`CREATE TABLE "addresses" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "title" character varying NOT NULL, "fullAddress" text NOT NULL, "comment" text, "isActive" boolean NOT NULL DEFAULT true, "sortOrder" integer NOT NULL DEFAULT '0', "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_745d8f43d3af10ab8247465e450" PRIMARY KEY ("id"))`);
        await queryRunner.query(`CREATE TABLE "banned_items" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "name" character varying NOT NULL, "description" text, "isActive" boolean NOT NULL DEFAULT true, "sortOrder" integer NOT NULL DEFAULT '0', "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_3fc9d912859099256083807b52b" PRIMARY KEY ("id"))`);
        await queryRunner.query(`CREATE TABLE "users" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "telegramId" bigint NOT NULL, "username" character varying, "firstName" character varying, "lastName" character varying, "phone" character varying, "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_a3ffb1c0c8416b9fc6f907b7433" PRIMARY KEY ("id"))`);
        await queryRunner.query(`CREATE UNIQUE INDEX "IDX_df18d17f84763558ac84192c75" ON "users" ("telegramId") `);
        await queryRunner.query(`CREATE TYPE "public"."parcels_status_enum" AS ENUM('created', 'in_transit', 'arrived', 'ready_for_pickup', 'issued', 'cancelled')`);
        await queryRunner.query(`CREATE TABLE "parcels" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "userId" uuid NOT NULL, "trackNumber" character varying NOT NULL, "description" character varying, "status" "public"."parcels_status_enum" NOT NULL DEFAULT 'created', "weightKg" character varying, "comment" text, "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_47847f79fee8a3926f2b3022a96" PRIMARY KEY ("id"))`);
        await queryRunner.query(`CREATE TABLE "support_info" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "message" text NOT NULL, "phone" character varying, "telegramContact" character varying, "email" character varying, "workingHours" character varying, "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_47994c0b5e73dd8848d1fe85569" PRIMARY KEY ("id"))`);
        await queryRunner.query(`ALTER TABLE "parcels" ADD CONSTRAINT "FK_47bf863dc29896d38741c4e282f" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE NO ACTION`);
    }

    public async down(queryRunner: QueryRunner): Promise<void> {
        await queryRunner.query(`ALTER TABLE "parcels" DROP CONSTRAINT "FK_47bf863dc29896d38741c4e282f"`);
        await queryRunner.query(`DROP TABLE "support_info"`);
        await queryRunner.query(`DROP TABLE "parcels"`);
        await queryRunner.query(`DROP TYPE "public"."parcels_status_enum"`);
        await queryRunner.query(`DROP INDEX "public"."IDX_df18d17f84763558ac84192c75"`);
        await queryRunner.query(`DROP TABLE "users"`);
        await queryRunner.query(`DROP TABLE "banned_items"`);
        await queryRunner.query(`DROP TABLE "addresses"`);
    }

}
