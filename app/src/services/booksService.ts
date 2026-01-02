import HttpService from './httpService';

export interface BookApiResponse {
  relPath: string;
  fileName: string;
  size: number;
  mtime: number;
  type: string;
}

export default class BooksService {
  static listBooks = async (): Promise<BookApiResponse[]> => {
    return await HttpService.getAsJson<BookApiResponse[]>('/api/v1/books');
  };
}
