import { useState, useEffect, useRef } from 'react'
import './App.css'
import SeriesPanel from './components/SeriesPanel'
import VolumePanel from './components/VolumePanel'
import ViewerPanel from './components/ViewerPanel'
import { usePanelResize } from './hooks/usePanelResize'
import type { VolumeInfo } from './types/wails'
import './types/wails.ts'

function App() {
  const [selectedSeries, setSelectedSeries] = useState('')
  const [selectedVolume, setSelectedVolume] = useState<VolumeInfo | null>(null)
  const [dataReady, setDataReady] = useState(false)
  const isInitialLoadRef = useRef(true)

  const { seriesWidth, volumeWidth, handleResizerMouseDown } = usePanelResize()

  useEffect(() => {
    const handleDataReady = async () => {
      console.log('dataReady event received')
      setDataReady(true)

      if (window.go?.app?.App?.GetLastSelection) {
        try {
          const selection = await window.go.app.App.GetLastSelection()
          console.log('Last selection:', selection)
          if (selection.series) {
            setSelectedSeries(selection.series)
          }
        } catch (error) {
          console.error('Failed to load last selection:', error)
        }
      }

      setTimeout(() => {
        isInitialLoadRef.current = false
      }, 200)
    }

    console.log('Setting up dataReady listener')
    if (window.runtime?.EventsOn) {
      window.runtime.EventsOn('dataReady', handleDataReady)
    } else {
      console.log('No runtime.EventsOn, calling handleDataReady directly')
      handleDataReady()
    }

    return () => {
      if (window.runtime?.EventsOff) {
        window.runtime.EventsOff('dataReady')
      }
    }
  }, [])

  useEffect(() => {
    if (!isInitialLoadRef.current && selectedSeries) {
      const saveSelection = async () => {
        if (window.go?.app?.App?.SaveLastSelection) {
          try {
            await window.go.app.App.SaveLastSelection(selectedSeries, selectedVolume?.name || '')
          } catch (error) {
            console.error('Failed to save selection:', error)
          }
        }
      }
      saveSelection()
    }
  }, [selectedSeries, selectedVolume])

  const handleSelectSeries = (series: string) => {
    setSelectedSeries(series)
    setSelectedVolume(null)
  }

  const handleSelectVolume = (volume: VolumeInfo) => {
    setSelectedVolume(volume)
  }

  const handleCloseViewer = () => {
    setSelectedVolume(null)
  }

  return (
    <div className="app">
      <div className="main-content">
        <div className="panel-wrapper" style={{ width: seriesWidth }}>
          <SeriesPanel
            selectedSeries={selectedSeries}
            dataReady={dataReady}
            onSelectSeries={handleSelectSeries}
            viewerOpen={!!selectedVolume}
          />
        </div>
        <div className="resizer" onMouseDown={(e) => handleResizerMouseDown('series', e)} />
        <div className="panel-wrapper" style={{ width: volumeWidth }}>
          <VolumePanel
            series={selectedSeries}
            selectedVolume={selectedVolume?.name || ''}
            onSelectVolume={handleSelectVolume}
          />
        </div>
        <div className="resizer" onMouseDown={(e) => handleResizerMouseDown('volume', e)} />
        <ViewerPanel volume={selectedVolume} onClose={handleCloseViewer} />
      </div>
    </div>
  )
}

export default App
