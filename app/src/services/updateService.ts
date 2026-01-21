import HttpService from './httpService';

export default class UpdateService {
  static async performUpdate(version: string): Promise<void> {
    await HttpService.post('/api/v1/update', { version });
  }
}
