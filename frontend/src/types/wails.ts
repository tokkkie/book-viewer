export {}

declare global {
  interface Window {
    go?: {
      app?: {
        App?: {
          GetSeriesList: () => Promise<string[]>
          GetVolumeList: (series: string) => Promise<VolumeInfo[]>
          GetZipContents: (zipPath: string) => Promise<VolumeInfo[]>
          GetImageList: (volumePath: string, isZip: boolean) => Promise<string[]>
          GetImageData: (volumePath: string, isZip: boolean, index: number) => Promise<ImageInfo>
          GetImageRange: (volumePath: string, isZip: boolean, startIndex: number, count: number) => Promise<ImageInfo[]>
          SetRootDirectory: (path: string) => Promise<void>
          GetRootDirectory: () => Promise<string>
          OpenDirectoryDialog: () => Promise<string>
          SaveLastSelection: (series: string, volume: string) => Promise<void>
          GetLastSelection: () => Promise<{ series: string; volume: string }>
        }
      }
    }
    runtime?: {
      EventsOn: (eventName: string, callback: (...args: any[]) => void) => void
      EventsOff: (eventName: string) => void
    }
  }
}

export interface VolumeInfo {
  name: string
  path: string
  isZip: boolean
  isDir: boolean
  images: number
  hasSubdirectories?: boolean
}

export interface ImageInfo {
  index: number
  name: string
  dataUrl: string
  mimeType: string
}
