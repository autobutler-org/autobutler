import HttpService from './httpService';

export interface Release {
  tagName: string;
  name: string;
  htmlUrl: string;
  publishedAt: string;
  isCurrentVersion: boolean;
}

export interface VersionResponse {
  semver: string;
  gitCommit: string;
  goVersion: string;
  buildDate: string;
}

export default class VersionService {
  static async doUpdate(version: string): Promise<void> {
    await HttpService.post('/api/v1/version/update', { version });
  }

  static getCurrentVersion = async (): Promise<string> => {
    try {
      const data =
        await HttpService.getAsJson<VersionResponse>('/api/v1/version');
      return data.semver || 'vX.Y.Z';
    } catch {
      return 'vX.Y.Z';
    }
  };

  static getAvailableReleases = async (): Promise<Release[]> => {
    try {
      const data = await HttpService.getAsJson<Release[]>(
        '/api/v1/version/available',
      );
      return data || [];
    } catch {
      return [];
    }
  };
}
