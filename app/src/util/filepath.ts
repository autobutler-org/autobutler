export const getFileNameFromPath = (path: string): string => {
  const segments = path.split('/');
  return segments[segments.length - 1] || '';
};

export const joinPaths = (
  basePath: string,
  ...additionalPaths: (string | null | undefined)[]
): string => [basePath, ...additionalPaths.filter((_) => _)].join('/');

export const joinPathsNormalized = (
  basePath: string,
  ...additionalPaths: (string | null | undefined)[]
): string => normalizePath(joinPaths(basePath, ...additionalPaths));

export const normalizePath = (path: string): string => {
  // Replace multiple slashes with a single slash
  let normalizedPath = path.replace(/\/+/g, '/');
  // Remove trailing slash if it's not the root "/"
  if (normalizedPath.length > 1 && normalizedPath.endsWith('/')) {
    normalizedPath = normalizedPath.slice(0, -1);
  }
  return normalizedPath;
};
