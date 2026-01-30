import { describe, expect, it } from 'vitest';
import { normalizePath } from './filepath';

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
