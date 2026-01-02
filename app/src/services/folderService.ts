import HttpService from './httpService';

export default class FolderService {
  static async createFolder(
    folderPath: string,
    folderName: string,
  ): Promise<void> {
    const formData = new FormData();
    formData.append('folderName', folderName);
    const url = folderPath
      ? `/api/v1/folder/cirrus/${folderPath}`
      : '/api/v1/folder/cirrus/';
    const response = await HttpService.postForm(url, formData);
    if (!response.ok) throw new Error('Failed to create folder');
  }
}
