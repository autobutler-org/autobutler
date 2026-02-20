import { describe, expect, it } from 'vitest';
import { formatBytes } from './bytes';

describe('bytesToHumanReadable', () => {
  it('should format bytes correctly', () => {
    expect(formatBytes(500)).toBe('500 B');
    expect(formatBytes(1024)).toBe('1.0 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
    expect(formatBytes(1048576)).toBe('1.0 MB');
    expect(formatBytes(1572864)).toBe('1.5 MB');
    expect(formatBytes(1073741824)).toBe('1.0 GB');
    expect(formatBytes(1610612736)).toBe('1.5 GB');
    expect(formatBytes(1099511627776)).toBe('1.0 TB');
    expect(formatBytes(1649267441664)).toBe('1.5 TB');
    expect(formatBytes(1125899906842624)).toBe('1.0 PB');
    expect(formatBytes(1688849860263936)).toBe('1.5 PB');
  });
});
