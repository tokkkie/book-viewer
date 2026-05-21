import { useState, useCallback, useEffect } from 'react'

export function usePanelResize() {
  const [seriesWidth, setSeriesWidth] = useState(() => {
    const saved = localStorage.getItem('panel:seriesWidth')
    return saved ? parseInt(saved, 10) : 250
  })

  const [volumeWidth, setVolumeWidth] = useState(() => {
    const saved = localStorage.getItem('panel:volumeWidth')
    return saved ? parseInt(saved, 10) : 300
  })

  useEffect(() => {
    localStorage.setItem('panel:seriesWidth', String(seriesWidth))
  }, [seriesWidth])

  useEffect(() => {
    localStorage.setItem('panel:volumeWidth', String(volumeWidth))
  }, [volumeWidth])

  const handleResizerMouseDown = useCallback((panel: 'series' | 'volume', e: React.MouseEvent) => {
    e.preventDefault()
    const startX = e.clientX
    const startWidth = panel === 'series' ? seriesWidth : volumeWidth

    const handleMouseMove = (e: MouseEvent) => {
      const delta = e.clientX - startX
      const newWidth = Math.max(150, Math.min(600, startWidth + delta))
      if (panel === 'series') {
        setSeriesWidth(newWidth)
      } else {
        setVolumeWidth(newWidth)
      }
    }

    const handleMouseUp = () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  }, [seriesWidth, volumeWidth])

  return {
    seriesWidth,
    volumeWidth,
    handleResizerMouseDown,
  }
}
