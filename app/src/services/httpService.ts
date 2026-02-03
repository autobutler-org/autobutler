import { joinPaths } from '@/util/filepath';

export default class HttpService {
  // NOTE: Can be empty in most of all cases, but is available in case you have a need for it
  static baseUrl: string = ``;

  static post = async (
    url: string,
    body?: unknown,
    options?: RequestInit,
  ): Promise<Response> =>
    HttpService._genericRequest(url, 'POST', body, options);

  static postForm = async (
    url: string,
    formData: FormData,
    options?: RequestInit,
  ): Promise<Response> => {
    const fetchOptions: RequestInit = {
      method: 'POST',
      body: formData,
      ...options,
    };
    return fetch(HttpService.constructUrl(url), fetchOptions);
  };

  static delete = async (
    url: string,
    options?: RequestInit,
  ): Promise<Response> =>
    HttpService._genericRequest(url, 'DELETE', undefined, options);

  static get = async (
    url: string,
    queryParams?: URLSearchParams,
    options?: RequestInit,
  ): Promise<Response> =>
    HttpService._genericRequest(
      queryParams ? url + `?${queryParams}` : url,
      'GET',
      undefined,
      options,
    );

  static getAsJson = async <T>(
    url: string,
    queryParams?: URLSearchParams,
    options?: RequestInit,
  ): Promise<T> =>
    HttpService._genericRequest(
      queryParams ? url + `?${queryParams}` : url,
      'GET',
      undefined,
      options,
    ).then(async (response: Response) => {
      return response.json() as Promise<T>;
    });

  static put = async (
    url: string,
    body: unknown,
    options?: RequestInit,
  ): Promise<Response> =>
    HttpService._genericRequest(url, 'PUT', body, options);

  private static constructUrl = (url: string): string => {
    while (url.startsWith('/')) {
      url = url.slice(1);
    }
    return joinPaths(HttpService.baseUrl, url);
  };

  private static _genericRequest = async (
    url: string,
    method: string,
    body?: unknown,
    options?: RequestInit,
  ): Promise<Response> => {
    const fetchOptions: RequestInit = {
      method,
      headers: {
        'Content-Type': 'application/json',
      },
      ...options,
    };

    if (body) {
      fetchOptions.body = JSON.stringify(body);
    }

    return fetch(HttpService.constructUrl(url), fetchOptions)
      .then((response) => {
        if (!response.ok) {
          return Promise.reject(response);
        }
        return response;
      })
      .catch((error) => {
        return Promise.reject(error);
      });
  };
}
