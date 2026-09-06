import { Post, PostLanguage, PostStatus } from '@app/database';
import { Injectable, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import sanitizeHtml from 'sanitize-html';
import { type FindOptionsWhere, Not, type Repository } from 'typeorm';
import type { CreatePostDto, UpdatePostDto } from './post.dto';

// Транслитерация кириллицы (ru + ky) для генерации слага из заголовка.
const TRANSLIT: Record<string, string> = {
  а: 'a',
  б: 'b',
  в: 'v',
  г: 'g',
  д: 'd',
  е: 'e',
  ё: 'e',
  ж: 'zh',
  з: 'z',
  и: 'i',
  й: 'y',
  к: 'k',
  л: 'l',
  м: 'm',
  н: 'n',
  о: 'o',
  п: 'p',
  р: 'r',
  с: 's',
  т: 't',
  у: 'u',
  ф: 'f',
  х: 'h',
  ц: 'ts',
  ч: 'ch',
  ш: 'sh',
  щ: 'sch',
  ъ: '',
  ы: 'y',
  ь: '',
  э: 'e',
  ю: 'yu',
  я: 'ya',
  ң: 'ng',
  ө: 'o',
  ү: 'u',
};

const SANITIZE_OPTIONS: sanitizeHtml.IOptions = {
  allowedTags: [
    'h1',
    'h2',
    'h3',
    'h4',
    'p',
    'a',
    'ul',
    'ol',
    'li',
    'blockquote',
    'strong',
    'em',
    's',
    'u',
    'code',
    'pre',
    'br',
    'hr',
    'img',
    'figure',
    'figcaption',
    'span',
  ],
  allowedAttributes: {
    a: ['href', 'target', 'rel'],
    img: ['src', 'alt', 'title'],
    span: ['class'],
  },
  allowedSchemes: ['http', 'https', 'mailto'],
  transformTags: {
    a: sanitizeHtml.simpleTransform('a', { rel: 'noopener noreferrer' }),
  },
};

function slugify(input: string): string {
  const base = input
    .toLowerCase()
    .split('')
    .map((ch) => (ch in TRANSLIT ? TRANSLIT[ch] : ch))
    .join('')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
  return base || `post-${Date.now()}`;
}

@Injectable()
export class PostsService {
  constructor(
    @InjectRepository(Post)
    private readonly repository: Repository<Post>,
  ) {}

  // ---- Публичное (только опубликованные) ----

  findPublished(language?: string): Promise<Post[]> {
    const where: FindOptionsWhere<Post> = { status: PostStatus.PUBLISHED };
    if (language && postIsLanguage(language)) {
      where.language = language as PostLanguage;
    }
    return this.repository.find({
      where,
      order: { publishedAt: 'DESC', createdAt: 'DESC' },
    });
  }

  async findPublishedBySlug(slug: string): Promise<Post> {
    const post = await this.repository.findOne({
      where: { slug, status: PostStatus.PUBLISHED },
    });
    if (!post) {
      throw new NotFoundException('Post not found');
    }
    return post;
  }

  // ---- Админ ----

  findAll(): Promise<Post[]> {
    return this.repository.find({ order: { createdAt: 'DESC' } });
  }

  async findOne(id: string): Promise<Post> {
    const post = await this.repository.findOne({ where: { id } });
    if (!post) {
      throw new NotFoundException('Post not found');
    }
    return post;
  }

  async create(dto: CreatePostDto): Promise<Post> {
    const slug = await this.uniqueSlug(dto.slug ?? slugify(dto.title));
    const status = (dto.status ?? 'draft') as PostStatus;
    const post = this.repository.create({
      title: dto.title,
      slug,
      excerpt: dto.excerpt ?? null,
      content: sanitizeHtml(dto.content, SANITIZE_OPTIONS),
      coverImageUrl: dto.coverImageUrl ?? null,
      language: (dto.language ?? 'ru') as PostLanguage,
      status,
      publishedAt: status === PostStatus.PUBLISHED ? new Date() : null,
    });
    return this.repository.save(post);
  }

  async update(id: string, dto: UpdatePostDto): Promise<Post> {
    const post = await this.findOne(id);

    if (dto.title !== undefined) post.title = dto.title;
    if (dto.slug !== undefined) post.slug = await this.uniqueSlug(dto.slug, id);
    if (dto.excerpt !== undefined) post.excerpt = dto.excerpt ?? null;
    if (dto.content !== undefined) post.content = sanitizeHtml(dto.content, SANITIZE_OPTIONS);
    if (dto.coverImageUrl !== undefined) post.coverImageUrl = dto.coverImageUrl ?? null;
    if (dto.language !== undefined) post.language = dto.language as PostLanguage;

    if (dto.status !== undefined) {
      const nextStatus = dto.status as PostStatus;
      if (nextStatus === PostStatus.PUBLISHED && post.status !== PostStatus.PUBLISHED) {
        post.publishedAt = post.publishedAt ?? new Date();
      }
      if (nextStatus === PostStatus.DRAFT) {
        post.publishedAt = null;
      }
      post.status = nextStatus;
    }

    return this.repository.save(post);
  }

  async remove(id: string): Promise<void> {
    const post = await this.findOne(id);
    await this.repository.remove(post);
  }

  private async uniqueSlug(desired: string, exceptId?: string): Promise<string> {
    const clean = slugify(desired);
    let candidate = clean;
    let n = 2;
    while (
      (await this.repository.count({
        where: exceptId ? { slug: candidate, id: Not(exceptId) } : { slug: candidate },
      })) > 0
    ) {
      candidate = `${clean}-${n++}`;
    }
    return candidate;
  }
}

function postIsLanguage(v: string): boolean {
  return v === 'ru' || v === 'ky' || v === 'en';
}
