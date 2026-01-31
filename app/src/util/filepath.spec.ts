import { describe, expect, it } from 'vitest';
import {
  getFileNameFromPath,
  joinPaths,
  joinPathsNormalized,
  normalizePath,
} from './filepath';

describe('getFileNameFromPath', () => {
  it('should return the file name from a simple path', () => {
    expect(getFileNameFromPath('foo/bar/baz.txt')).toBe('baz.txt');
  });

  it('should return the file name when there is no slash', () => {
    expect(getFileNameFromPath('baz.txt')).toBe('baz.txt');
  });

  it('should return empty string for empty path', () => {
    expect(getFileNameFromPath('')).toBe('');
  });

  it('should return empty string for path ending with slash', () => {
    expect(getFileNameFromPath('foo/bar/')).toBe('');
  });

  it('should return the file name for root file', () => {
    expect(getFileNameFromPath('/baz.txt')).toBe('baz.txt');
  });

  it('should return empty string for only slashes', () => {
    expect(getFileNameFromPath('/')).toBe('');
    expect(getFileNameFromPath('////')).toBe('');
  });

  it('should handle file name with spaces', () => {
    expect(getFileNameFromPath('foo/bar/my file.txt')).toBe('my file.txt');
  });

  it('should handle hidden files', () => {
    expect(getFileNameFromPath('foo/.env')).toBe('.env');
  });

  it('should handle path with multiple consecutive slashes', () => {
    expect(getFileNameFromPath('foo//bar///baz.txt')).toBe('baz.txt');
  });
});

describe('joinPaths', () => {
  it('should join two simple paths', () => {
    expect(joinPaths('foo', 'bar')).toBe('foo/bar');
  });

  it('should join multiple paths', () => {
    expect(joinPaths('foo', 'bar', 'baz')).toBe('foo/bar/baz');
  });

  it('should ignore null and undefined values', () => {
    expect(joinPaths('foo', null, 'bar', undefined, 'baz')).toBe('foo/bar/baz');
  });

  it('should handle empty additional paths', () => {
    expect(joinPaths('foo')).toBe('foo');
  });

  it('should handle basePath with trailing slash', () => {
    expect(joinPaths('foo/', 'bar')).toBe('foo//bar');
  });

  it('should handle additional paths with leading slash', () => {
    expect(joinPaths('foo', '/bar')).toBe('foo//bar');
  });

  it('should handle all empty strings', () => {
    expect(joinPaths('', '', '')).toBe('');
  });

  it('should handle basePath as empty string', () => {
    expect(joinPaths('', 'bar')).toBe('/bar');
  });
});

describe('joinPathsNormalized', () => {
  it('should join and normalize simple paths', () => {
    expect(joinPathsNormalized('foo', 'bar')).toBe('foo/bar');
  });

  it('should join and normalize multiple paths', () => {
    expect(joinPathsNormalized('foo', 'bar', 'baz')).toBe('foo/bar/baz');
  });

  it('should ignore null and undefined and normalize', () => {
    expect(joinPathsNormalized('foo', null, 'bar', undefined, 'baz')).toBe(
      'foo/bar/baz',
    );
  });

  it('should normalize multiple slashes', () => {
    expect(joinPathsNormalized('foo/', '/bar//', 'baz/')).toBe('foo/bar/baz');
  });

  it('should handle basePath as empty string', () => {
    expect(joinPathsNormalized('', 'bar')).toBe('/bar');
  });

  it('should handle all empty strings', () => {
    expect(joinPathsNormalized('', '', '')).toBe('');
  });

  it('should handle only basePath', () => {
    expect(joinPathsNormalized('foo')).toBe('foo');
  });

  it('should handle only slashes', () => {
    expect(joinPathsNormalized('////', '///')).toBe('/');
  });
});

describe('normalizePath', () => {
  it('should replace multiple slashes with a single slash', () => {
    expect(normalizePath('foo//bar///baz')).toBe('foo/bar/baz');
  });

  it('should remove trailing slash if not root', () => {
    expect(normalizePath('foo/bar/')).toBe('foo/bar');
  });

  it('should not remove trailing slash for root', () => {
    expect(normalizePath('/')).toBe('/');
  });

  it('should handle empty string', () => {
    expect(normalizePath('')).toBe('');
  });

  it('should handle only slashes', () => {
    expect(normalizePath('////')).toBe('/');
  });

  it('should handle path with leading slash', () => {
    expect(normalizePath('/foo/bar/')).toBe('/foo/bar');
  });

  it('should handle path with no slashes', () => {
    expect(normalizePath('foobar')).toBe('foobar');
  });

  it('should handle path with spaces and slashes', () => {
    expect(normalizePath(' /foo//bar/ ')).toBe(' /foo/bar/ ');
  });

  it('should handle path ending with multiple slashes', () => {
    expect(normalizePath('foo/bar///')).toBe('foo/bar');
  });

  it('should replace multiple slashes with a single slash', () => {
    expect(normalizePath('foo//bar///baz')).toBe('foo/bar/baz');
  });

  it('should remove trailing slash if not root', () => {
    expect(normalizePath('foo/bar/')).toBe('foo/bar');
  });

  it('should not remove trailing slash for root', () => {
    expect(normalizePath('/')).toBe('/');
  });

  it('should handle empty string', () => {
    expect(normalizePath('')).toBe('');
  });

  it('should handle only slashes', () => {
    expect(normalizePath('////')).toBe('/');
  });

  it('should handle path with leading slash', () => {
    expect(normalizePath('/foo/bar/')).toBe('/foo/bar');
  });

  it('should handle path with no slashes', () => {
    expect(normalizePath('foobar')).toBe('foobar');
  });

  it('should handle path with spaces and slashes', () => {
    expect(normalizePath(' /foo//bar/ ')).toBe(' /foo/bar/ ');
  });

  it('should handle path ending with multiple slashes', () => {
    expect(normalizePath('foo/bar///')).toBe('foo/bar');
  });
});
