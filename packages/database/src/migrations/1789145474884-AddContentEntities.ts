import { MigrationInterface, QueryRunner } from "typeorm";

export class AddContentEntities1789145474884 implements MigrationInterface {
    name = 'AddContentEntities1789145474884'

    public async up(queryRunner: QueryRunner): Promise<void> {
        await queryRunner.query(`CREATE TYPE "public"."blog_posts_locale_enum" AS ENUM('ru', 'ky', 'en', 'de')`);
        await queryRunner.query(`CREATE TABLE "blog_posts" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "locale" "public"."blog_posts_locale_enum" NOT NULL, "slug" character varying NOT NULL, "title" character varying NOT NULL, "excerpt" text, "content" text NOT NULL, "coverImageUrl" character varying, "seoTitle" character varying, "seoDescription" character varying, "isPublished" boolean NOT NULL DEFAULT false, "publishedAt" TIMESTAMP, "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_dd2add25eac93daefc93da9d387" PRIMARY KEY ("id"))`);
        await queryRunner.query(`CREATE UNIQUE INDEX "IDX_69800603022eb38334c3ba4917" ON "blog_posts" ("locale", "slug") `);
        await queryRunner.query(`CREATE TABLE "gallery_items" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "imageUrl" character varying NOT NULL, "caption" character varying, "altText" character varying, "sortOrder" integer NOT NULL DEFAULT '0', "isPublished" boolean NOT NULL DEFAULT true, "createdAt" TIMESTAMP NOT NULL DEFAULT now(), "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_ca2915427d004dec2ff17f45a49" PRIMARY KEY ("id"))`);
        await queryRunner.query(`CREATE TYPE "public"."page_seo_page_enum" AS ENUM('home', 'blog', 'gallery')`);
        await queryRunner.query(`CREATE TYPE "public"."page_seo_locale_enum" AS ENUM('ru', 'ky', 'en', 'de')`);
        await queryRunner.query(`CREATE TABLE "page_seo" ("id" uuid NOT NULL DEFAULT uuid_generate_v4(), "page" "public"."page_seo_page_enum" NOT NULL, "locale" "public"."page_seo_locale_enum" NOT NULL, "title" character varying, "description" character varying, "ogImageUrl" character varying, "updatedAt" TIMESTAMP NOT NULL DEFAULT now(), CONSTRAINT "PK_17c2a2543029a59bab0ca323f1b" PRIMARY KEY ("id"))`);
        await queryRunner.query(`CREATE UNIQUE INDEX "IDX_1b84a12886536e65d1003b1311" ON "page_seo" ("page", "locale") `);
    }

    public async down(queryRunner: QueryRunner): Promise<void> {
        await queryRunner.query(`DROP INDEX "public"."IDX_1b84a12886536e65d1003b1311"`);
        await queryRunner.query(`DROP TABLE "page_seo"`);
        await queryRunner.query(`DROP TYPE "public"."page_seo_locale_enum"`);
        await queryRunner.query(`DROP TYPE "public"."page_seo_page_enum"`);
        await queryRunner.query(`DROP TABLE "gallery_items"`);
        await queryRunner.query(`DROP INDEX "public"."IDX_69800603022eb38334c3ba4917"`);
        await queryRunner.query(`DROP TABLE "blog_posts"`);
        await queryRunner.query(`DROP TYPE "public"."blog_posts_locale_enum"`);
    }

}
