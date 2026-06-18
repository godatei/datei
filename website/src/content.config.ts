import {glob} from 'astro/loaders';
import {z} from 'astro/zod';
import {defineCollection, type SchemaContext} from 'astro:content';

export const BlogPostConfigSchema = ({image}: SchemaContext) =>
  z.object({
    title: z.string(),
    description: z.string(),
    publishDate: z.coerce.date(),
    lastUpdated: z.coerce.date(),
    slug: z.string(),
    authors: z.array(
      z.object({
        name: z.string(),
        role: z.string(),
        image: image(),
        linkedIn: z.string(),
        gitHub: z.string(),
      }),
    ),
    image: image(),
    tags: z.array(z.string()),
  });

export const DocConfigSchema = z.object({
  title: z.string(),
  description: z.string(),
  sidebar: z
    .object({
      label: z.string().optional(),
      order: z.number().optional(),
    })
    .default({}),
});

export const collections = {
  docs: defineCollection({
    loader: glob({pattern: '**/*.{md,mdx}', base: 'src/content/docs'}),
    schema: DocConfigSchema,
  }),
  blog: defineCollection({
    loader: glob({pattern: '**/*.{md,mdx}', base: 'src/content/blog'}),
    schema: BlogPostConfigSchema,
  }),
};

export type BlogPostConfig = z.output<ReturnType<typeof BlogPostConfigSchema>>;
export type DocConfig = z.output<typeof DocConfigSchema>;
