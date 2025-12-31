import HttpService from './httpService'

export default class TextFileService {
  static async fetchTextFile(url: string): Promise<string> {
    // Use fetch directly for text, but through HttpService for consistent baseUrl
    const response = await HttpService.get(url)
    if (!response.ok) throw new Error('Failed to load file')
    return await response.text()
  }
}
