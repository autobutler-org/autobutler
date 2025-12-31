export default class HttpService {
  static baseUrl: string = `${window.location.protocol}//${window.location.hostname}:8080`

  static get = async (url: string, options?: RequestInit): Promise<Response> =>
    this._genericRequest<Response>(url, 'GET', undefined, options)

  static getAsJson = async <T>(
    url: string,
    options?: RequestInit,
  ): Promise<T> => this._genericRequest<T>(url, 'GET', undefined, options)

  private static constructUrl(url: string): string {
    while (url.startsWith('/')) {
      url = url.slice(1)
    }
    return `${this.baseUrl}/${url}`
  }

  private static _genericRequest = async <T>(
    url: string,
    method: string,
    body?: never,
    options?: RequestInit,
  ): Promise<T> => {
    const fetchOptions: RequestInit = {
      method,
      headers: {
        'Content-Type': 'application/json',
      },
      ...options,
    }

    if (body) {
      fetchOptions.body = JSON.stringify(body)
    }

    return fetch(HttpService.constructUrl(url), fetchOptions)
      .then((response) => {
        if (!response.ok) {
          return Promise.reject(response)
        }
        return response as T
      })
      .catch((error) => {
        return Promise.reject(error)
      })
  }
}
