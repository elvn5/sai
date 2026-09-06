import { MigrationInterface, QueryRunner } from "typeorm";

export class AddPosts1788600000000 implements MigrationInterface {
    name = 'AddPosts1788600000000'

    public async up(queryRunner: QueryRunner): Promise<void> {
        await queryRunner.query(`CREATE TYPE "public"."posts_language_enum" AS ENUM('ru', 'ky', 'en')`);
        await queryRunner.query(`CREATE TYPE "public"."posts_status_enum" AS ENUM('draft', 'published')`);
        await queryRunner.query(`CREATE TABLE "posts" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "slug" character varying NOT NULL, "title" character varying NOT NULL, "excerpt" text, "content" text NOT NULL, "coverImageUrl" character varying, "language" "public"."posts_language_enum" NOT NULL DEFAULT 'ru', "status" "public"."posts_status_enum" NOT NULL DEFAULT 'draft', "publishedAt" TIMESTAMP WITH TIME ZONE, "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_posts_id" PRIMARY KEY ("id"))`);
        await queryRunner.query(`CREATE UNIQUE INDEX "IDX_posts_slug" ON "posts" ("slug")`);
    }

    public async down(queryRunner: QueryRunner): Promise<void> {
        await queryRunner.query(`DROP INDEX "public"."IDX_posts_slug"`);
        await queryRunner.query(`DROP TABLE "posts"`);
        await queryRunner.query(`DROP TYPE "public"."posts_status_enum"`);
        await queryRunner.query(`DROP TYPE "public"."posts_language_enum"`);
    }
}
