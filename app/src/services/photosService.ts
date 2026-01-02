import HttpService from './httpService';

export interface PhotoApiResponse {
  relPath: string;
  fileName: string;
  size: number;
  mtime: string;
}

export default class PhotosService {
  static listPhotos = async (): Promise<PhotoApiResponse[]> => {
    return await HttpService.getAsJson<PhotoApiResponse[]>('/api/v1/photos');
  };
}
