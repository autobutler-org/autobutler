import { describe, it, expect } from 'vitest';
import { arrayCompare, arrayEquals } from './array';

describe('arrayCompare', () => {
  const numCmp = (a: number, b: number) => a - b;

  it('returns 0 for two undefined arrays', () => {
    expect(arrayCompare(undefined, undefined, numCmp)).toBe(0);
  });

  it('returns -1 if first array is undefined', () => {
    expect(arrayCompare(undefined, [1, 2], numCmp)).toBe(-1);
  });

  it('returns 1 if second array is undefined', () => {
    expect(arrayCompare([1, 2], undefined, numCmp)).toBe(1);
  });

  it('returns 0 for equal arrays', () => {
    expect(arrayCompare([1, 2, 3], [1, 2, 3], numCmp)).toBe(0);
  });

  it('returns -1 if first array is less', () => {
    expect(arrayCompare([1, 2], [1, 2, 3], numCmp)).toBe(-1);
  });

  it('returns 1 if first array is greater', () => {
    expect(arrayCompare([1, 2, 3], [1, 2], numCmp)).toBe(1);
  });

  it('returns correct comparison for differing elements', () => {
    expect(arrayCompare([1, 2, 4], [1, 2, 3], numCmp)).toBe(1);
    expect(arrayCompare([1, 2, 2], [1, 2, 3], numCmp)).toBe(-1);
  });
});

describe('arrayEquals', () => {
  const eq = (a: number, b: number) => a === b;

  it('returns true for two undefined arrays', () => {
    expect(arrayEquals(undefined, undefined, eq)).toBe(true);
  });

  it('returns false if one array is undefined', () => {
    expect(arrayEquals(undefined, [1], eq)).toBe(false);
    expect(arrayEquals([1], undefined, eq)).toBe(false);
  });

  it('returns false for arrays of different lengths', () => {
    expect(arrayEquals([1, 2], [1, 2, 3], eq)).toBe(false);
  });

  it('returns true for equal arrays', () => {
    expect(arrayEquals([1, 2, 3], [1, 2, 3], eq)).toBe(true);
  });

  it('returns false for arrays with different elements', () => {
    expect(arrayEquals([1, 2, 3], [1, 2, 4], eq)).toBe(false);
  });
});

describe('arrayCompare with default comparators', () => {
  it('compares number arrays with default comparator', () => {
    const cmp = (a: number, b: number) => a - b;
    expect(arrayCompare([1, 2, 3], [1, 2, 3], cmp)).toBe(0);
    expect(arrayCompare([1, 2, 2], [1, 2, 3], cmp)).toBe(-1);
    expect(arrayCompare([1, 2, 4], [1, 2, 3], cmp)).toBe(1);
  });

  it('compares string arrays with default comparator', () => {
    const cmp = (a: string, b: string) => a.localeCompare(b);
    expect(arrayCompare(['a', 'b', 'c'], ['a', 'b', 'c'], cmp)).toBe(0);
    expect(arrayCompare(['a', 'b', 'b'], ['a', 'b', 'c'], cmp)).toBe(-1);
    expect(arrayCompare(['a', 'b', 'd'], ['a', 'b', 'c'], cmp)).toBe(1);
  });
});
