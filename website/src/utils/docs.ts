import {getCollection, type CollectionEntry} from 'astro:content';

export type Doc = CollectionEntry<'docs'>;

export const getSortedDocs = async (): Promise<Array<Doc>> => {
  return (await getCollection('docs')).sort((a, b) => {
    const orderA = a.data.sidebar.order ?? Number.MAX_SAFE_INTEGER;
    const orderB = b.data.sidebar.order ?? Number.MAX_SAFE_INTEGER;
    if (orderA !== orderB) {
      return orderA - orderB;
    }
    return a.data.title.localeCompare(b.data.title);
  });
};

export const getDocHref = (doc: Doc): string =>
  doc.id === 'index' ? '/docs/' : `/docs/${doc.id}/`;

export const getDocLabel = (doc: Doc): string =>
  doc.data.sidebar.label ?? doc.data.title;
