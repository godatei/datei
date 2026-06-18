import {getCollection, type CollectionEntry} from 'astro:content';

export type Post = CollectionEntry<'blog'>;

export const getSortedPosts = async (): Promise<Array<Post>> => {
  return (await getCollection('blog')).sort((a, b) => {
    const dateA = a.data.publishDate
      ? new Date(a.data.publishDate).valueOf()
      : 0;
    const dateB = b.data.publishDate
      ? new Date(b.data.publishDate).valueOf()
      : 0;
    return dateB - dateA;
  });
};
