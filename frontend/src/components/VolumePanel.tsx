import { useState, useEffect, useCallback, useRef } from 'react'
import type { VolumeInfo } from '../types/wails'

interface VolumePanelProps {
  series: string
  selectedVolume: string
  onSelectVolume: (volume: VolumeInfo) => void
}

export default function VolumePanel({ series, selectedVolume, onSelectVolume }: VolumePanelProps) {
  const [volumeList, setVolumeList] = useState<VolumeInfo[]>([])
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const [currentPath, setCurrentPath] = useState<string>('')
  const [pathStack, setPathStack] = useState<string[]>([])

  useEffect(() => {
    if (!series) {
      setVolumeList([])
      setSelectedIndex(-1)
      setPathStack([])
      setCurrentPath('')
      return
    }

    // Reset navigation when series changes
    setPathStack([])
    setCurrentPath('')

    const loadVolumes = async () => {
      if (window.go?.app?.App?.GetVolumeList) {
        try {
          const list = await window.go.app.App.GetVolumeList(series)
          setVolumeList(list || [])
        } catch (error) {
          console.error('Failed to load volumes:', error)
          setVolumeList([])
        }
      }
    }

    loadVolumes()
  }, [series])

  // Update selected index when selectedVolume changes
  useEffect(() => {
    if (selectedVolume && volumeList.length > 0) {
      const idx = volumeList.findIndex(v => v.name === selectedVolume)
      setSelectedIndex(idx)
    }
  }, [selectedVolume, volumeList])

  // When viewer is closed (selectedVolume becomes empty), reset navigation if we're inside a zip
  const prevSelectedVolumeRef = useRef(selectedVolume)
  useEffect(() => {
    // Only reset if viewer was open and is now closed
    if (prevSelectedVolumeRef.current && !selectedVolume && pathStack.length > 0) {
      console.log('Viewer closed, resetting navigation')
      setPathStack([])
      setCurrentPath('')
      // Reload series volumes
      if (series && window.go?.app?.App?.GetVolumeList) {
        window.go.app.App.GetVolumeList(series).then(list => {
          setVolumeList(list || [])
        }).catch(error => {
          console.error('Failed to reload volumes:', error)
        })
      }
    }
    prevSelectedVolumeRef.current = selectedVolume
  }, [selectedVolume, pathStack.length, series])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    // Disable keyboard navigation when viewer is open
    if (selectedVolume || volumeList.length === 0) return

    if (e.key === 'ArrowUp') {
      e.preventDefault()
      const newIndex = Math.max(0, selectedIndex - 1)
      setSelectedIndex(newIndex)
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      const newIndex = Math.min(volumeList.length - 1, selectedIndex + 1)
      setSelectedIndex(newIndex)
    } else if (e.key === 'Enter' && selectedIndex >= 0) {
      onSelectVolume(volumeList[selectedIndex])
    }
  }, [volumeList, selectedIndex, onSelectVolume, selectedVolume])

  const handleClick = (index: number) => {
    setSelectedIndex(index)
  }

  const handleDoubleClick = async (volume: VolumeInfo) => {
    console.log('Double click on volume:', volume)
    console.log('  isZip:', volume.isZip, 'images:', volume.images, 'hasSubdirectories:', volume.hasSubdirectories)
    console.log('  path:', volume.path)
    
    // If it's a zip file with subdirectories, explore it
    if (volume.isZip && volume.hasSubdirectories && window.go?.app?.App?.GetZipContents) {
      console.log('  -> Zip has subdirectories, checking contents...')
      try {
        const contents = await window.go.app.App.GetZipContents(volume.path)
        console.log('  -> Got contents:', contents)
        if (contents && contents.length > 0) {
          // Has nested content, navigate into it
          console.log('  -> Navigating into zip')
          setPathStack(prev => [...prev, currentPath || series])
          setCurrentPath(volume.path)
          setVolumeList(contents)
          setSelectedIndex(-1)
          return
        }
      } catch (error) {
        console.error('Failed to load zip contents:', error)
      }
    }
    
    // Otherwise, open as viewer (including directories inside zips)
    console.log('  -> Opening viewer')
    onSelectVolume(volume)
  }

  const handleGoBack = async () => {
    if (pathStack.length === 0) return

    const previousPath = pathStack[pathStack.length - 1]
    setPathStack(prev => prev.slice(0, -1))
    setCurrentPath(previousPath)

    console.log('Going back to:', previousPath || series)

    // Reload the previous level
    if (previousPath && previousPath.includes('::')) {
      // Going back inside a zip
      if (window.go?.app?.App?.GetZipContents) {
        try {
          const list = await window.go.app.App.GetZipContents(previousPath)
          setVolumeList(list || [])
        } catch (error) {
          console.error('Failed to load zip contents:', error)
        }
      }
    } else if (window.go?.app?.App?.GetVolumeList) {
      // Going back to series level
      try {
        const list = await window.go.app.App.GetVolumeList(previousPath || series)
        setVolumeList(list || [])
      } catch (error) {
        console.error('Failed to load volumes:', error)
      }
    }
  }

  return (
    <div className="panel volume-panel" onKeyDown={handleKeyDown} tabIndex={0}>
      <div className="panel-header">
        <h2>Volumes</h2>
        {pathStack.length > 0 && (
          <button onClick={handleGoBack} className="btn-back">← Back</button>
        )}
      </div>
      <div className="panel-content">
        {volumeList && volumeList.map((volume, index) => (
          <div
            key={volume.path}
            className={`list-item ${index === selectedIndex ? 'selected' : ''}`}
            onClick={() => handleClick(index)}
            onDoubleClick={() => handleDoubleClick(volume)}
          >
            <div className="volume-name">{volume.name}</div>
            <div className="volume-info">{volume.images > 0 ? `${volume.images} images` : volume.isZip ? 'Archive' : 'Folder'}</div>
          </div>
        ))}
      </div>
    </div>
  )
}
