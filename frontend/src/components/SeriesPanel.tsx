import { useState, useEffect, useCallback } from 'react'

interface SeriesPanelProps {
  selectedSeries: string
  dataReady: boolean
  onSelectSeries: (series: string) => void
  viewerOpen: boolean
}

export default function SeriesPanel({ selectedSeries, dataReady, onSelectSeries, viewerOpen }: SeriesPanelProps) {
  const [seriesList, setSeriesList] = useState<string[]>([])
  const [selectedIndex, setSelectedIndex] = useState(-1)

  useEffect(() => {
    const loadSeries = async () => {
      console.log('SeriesPanel: Attempting to load series, dataReady:', dataReady)
      if (window.go?.app?.App?.GetRootDirectory && window.go?.app?.App?.GetSeriesList) {
        try {
          const rootPath = await window.go.app.App.GetRootDirectory()
          console.log('Root directory loaded:', rootPath)
          if (rootPath) {
            const list = await window.go.app.App.GetSeriesList()
            console.log('Series list loaded:', list.length, 'items')
            setSeriesList(list || [])
            if (selectedSeries) {
              const idx = list.indexOf(selectedSeries)
              setSelectedIndex(idx)
            }
          } else {
            console.log('No root directory configured')
          }
        } catch (error) {
          console.error('Failed to load series:', error)
        }
      } else {
        console.log('Wails API not ready yet')
      }
    }

    // dataReadyを待たずに、少し遅延させて実行
    const timer = setTimeout(loadSeries, 500)
    return () => clearTimeout(timer)
  }, [])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    // Disable keyboard navigation when viewer is open
    if (viewerOpen || seriesList.length === 0) return

    if (e.key === 'ArrowUp') {
      e.preventDefault()
      const newIndex = Math.max(0, selectedIndex - 1)
      setSelectedIndex(newIndex)
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      const newIndex = Math.min(seriesList.length - 1, selectedIndex + 1)
      setSelectedIndex(newIndex)
    } else if (e.key === 'Enter' && selectedIndex >= 0) {
      onSelectSeries(seriesList[selectedIndex])
    }
  }, [seriesList, selectedIndex, onSelectSeries, viewerOpen])

  const handleClick = (series: string, index: number) => {
    setSelectedIndex(index)
    onSelectSeries(series)
  }

  const handleSetRoot = async () => {
    if (window.go?.app?.App?.OpenDirectoryDialog) {
      try {
        const path = await window.go.app.App.OpenDirectoryDialog()
        if (path) {
          await window.go.app.App.SetRootDirectory(path)
          const list = await window.go.app.App.GetSeriesList()
          setSeriesList(list)
        }
      } catch (error) {
        console.error('Failed to set root:', error)
      }
    }
  }

  return (
    <div className="panel series-panel" onKeyDown={handleKeyDown} tabIndex={0}>
      <div className="panel-header">
        <h2>Series</h2>
        <button onClick={handleSetRoot} className="btn-set-root" title="Set Root Directory">⚙️</button>
      </div>
      <div className="panel-content">
        {seriesList && seriesList.map((series, index) => (
          <div
            key={series}
            className={`list-item ${index === selectedIndex ? 'selected' : ''}`}
            onClick={() => handleClick(series, index)}
          >
            {series}
          </div>
        ))}
      </div>
    </div>
  )
}
