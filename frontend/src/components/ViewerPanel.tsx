import { useState, useEffect, useCallback, useRef } from 'react'
import type { VolumeInfo, ImageInfo } from '../types/wails'

interface ViewerPanelProps {
  volume: VolumeInfo | null
  onClose: () => void
}

export default function ViewerPanel({ volume, onClose }: ViewerPanelProps) {
  const [images, setImages] = useState<ImageInfo[]>([])
  const [currentIndex, setCurrentIndex] = useState(0)
  const [scale, setScale] = useState(1)
  const [translate, setTranslate] = useState({ x: 0, y: 0 })
  const [isDragging, setIsDragging] = useState(false)
  const dragStart = useRef({ x: 0, y: 0 })
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!volume) {
      setImages([])
      setCurrentIndex(0)
      setScale(1)
      setTranslate({ x: 0, y: 0 })
      return
    }

    const loadImages = async () => {
      if (window.go?.app?.App?.GetImageList) {
        try {
          const list = await window.go.app.App.GetImageList(volume.path, volume.isZip)
          const imageInfos: ImageInfo[] = list.map((name, index) => ({
            index,
            name,
            dataUrl: '',
            mimeType: '',
          }))
          setImages(imageInfos)
          setCurrentIndex(0)
          loadImageData(volume, 0)
        } catch (error) {
          console.error('Failed to load images:', error)
        }
      }
    }

    loadImages()
  }, [volume])

  const loadImageData = async (vol: VolumeInfo, index: number) => {
    if (window.go?.app?.App?.GetImageRange) {
      try {
        const range = await window.go.app.App.GetImageRange(vol.path, vol.isZip, index, 3)
        setImages(prev => {
          const newImages = [...prev]
          range.forEach(img => {
            if (img.index < newImages.length) {
              newImages[img.index] = img
            }
          })
          return newImages
        })
      } catch (error) {
        console.error('Failed to load image data:', error)
      }
    }
  }

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (!volume) return

    if (e.key === 'Escape') {
      onClose()
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault()
      setCurrentIndex(prev => {
        const next = Math.min(images.length - 1, prev + 1)
        if (next !== prev && volume) {
          loadImageData(volume, next)
        }
        return next
      })
      setTranslate({ x: 0, y: 0 })
    } else if (e.key === 'ArrowRight') {
      e.preventDefault()
      setCurrentIndex(prev => {
        const next = Math.max(0, prev - 1)
        if (next !== prev && volume) {
          loadImageData(volume, next)
        }
        return next
      })
      setTranslate({ x: 0, y: 0 })
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      setCurrentIndex(prev => {
        const next = Math.min(images.length - 1, prev + 2)
        if (next !== prev && volume) {
          loadImageData(volume, next)
        }
        return next
      })
      setTranslate({ x: 0, y: 0 })
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setCurrentIndex(prev => {
        const next = Math.max(0, prev - 2)
        if (next !== prev && volume) {
          loadImageData(volume, next)
        }
        return next
      })
      setTranslate({ x: 0, y: 0 })
    }
  }, [volume, images.length, onClose])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault()
    setScale(prev => Math.max(0.5, Math.min(5, prev - e.deltaY * 0.001)))
  }

  const handleMouseDown = (e: React.MouseEvent) => {
    if (scale > 1) {
      setIsDragging(true)
      dragStart.current = { x: e.clientX - translate.x, y: e.clientY - translate.y }
    }
  }

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging) {
      setTranslate({
        x: e.clientX - dragStart.current.x,
        y: e.clientY - dragStart.current.y,
      })
    }
  }

  const handleMouseUp = () => {
    setIsDragging(false)
  }

  if (!volume) {
    return (
      <div className="panel viewer-panel empty">
        <div className="empty-message">Select a volume to view</div>
      </div>
    )
  }

  const isCover = currentIndex === 0
  const currentImage = images[currentIndex]
  const nextImage = !isCover && currentIndex + 1 < images.length ? images[currentIndex + 1] : null

  return (
    <div
      ref={containerRef}
      className="panel viewer-panel"
      onWheel={handleWheel}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
    >
      <div className="viewer-header">
        <div className="viewer-info">
          {volume.name} - Page {currentIndex + 1} / {images.length}
        </div>
        <button onClick={onClose} className="btn-close">Close (Esc)</button>
      </div>
      <div className="viewer-content">
        {isCover ? (
          <div 
            className="image-container single"
            style={{
              transform: `scale(${scale}) translate(${translate.x / scale}px, ${translate.y / scale}px)`,
              cursor: scale > 1 ? 'move' : 'default',
            }}
          >
            <div className="image-wrapper">
              {currentImage?.dataUrl && (
                <img
                  src={currentImage.dataUrl}
                  alt={currentImage.name}
                />
              )}
            </div>
          </div>
        ) : (
          <div 
            className="image-container double"
            style={{
              transform: `scale(${scale}) translate(${translate.x / scale}px, ${translate.y / scale}px)`,
              cursor: scale > 1 ? 'move' : 'default',
            }}
          >
            <div className="image-wrapper">
              {currentImage?.dataUrl && (
                <img
                  src={currentImage.dataUrl}
                  alt={currentImage.name}
                />
              )}
            </div>
            <div className="image-wrapper">
              {nextImage?.dataUrl && (
                <img
                  src={nextImage.dataUrl}
                  alt={nextImage.name}
                />
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
