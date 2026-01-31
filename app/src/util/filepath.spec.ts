import { describe, expect, it } from 'vitest';
import { getFileNameFromPath, normalizePath } from './filepath';

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
});

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
