import type { ReactElement } from 'react'
import { useCallback, useEffect, useState } from 'react'
import Screenshots from './Screenshots'
import type { Bounds } from './Screenshots/types'

export interface Display {
  id: number
  x: number
  y: number
  width: number
  height: number
}

async function handleOk(blob: Blob | null, bounds: Bounds) {
  if (!blob) return
  const arrayBuffer = await blob.arrayBuffer()
  const uint8Array = new Uint8Array(arrayBuffer)
  const base64 = btoa(Array.from(uint8Array).map((b) => String.fromCharCode(b)).join(''))
  const dataUrl = `data:image/png;base64,${base64}`

  if ((window as any).__TAURI_INTERNALS__) {
    const { invoke } = await import('@tauri-apps/api/core')
    invoke('ok_screenshot_overlay', { dataUrl, boundsJson: JSON.stringify(bounds) })
  } else if (window.go?.main?.App) {
    window.go.main.App.CompleteScreenshot(dataUrl, JSON.stringify(bounds))
  }
}

async function handleCancel() {
  if ((window as any).__TAURI_INTERNALS__) {
    const { invoke } = await import('@tauri-apps/api/core')
    invoke('cancel_screenshot_overlay')
  } else if (window.go?.main?.App) {
    window.go.main.App.CancelScreenshot()
  }
}

async function handleSave(blob: Blob | null, bounds: Bounds) {
  if (!blob) return
  const arrayBuffer = await blob.arrayBuffer()
  const uint8Array = new Uint8Array(arrayBuffer)

  if ((window as any).__TAURI_INTERNALS__) {
    const { invoke } = await import('@tauri-apps/api/core')
    invoke('save_screenshot_overlay', { data: Array.from(uint8Array), boundsJson: JSON.stringify(bounds) })
  } else if (window.go?.main?.App) {
    window.go.main.App.SaveScreenshot(uint8Array, JSON.stringify(bounds))
  }
}

interface ScreenshotData {
  imageUrl?: string
  displayInfo?: string
}

function getInjectedData(): ScreenshotData | null {
  const data = (window as any).__SCREENSHOT_DATA__ as ScreenshotData | undefined
  if (data?.imageUrl) return data
  return null
}

export default function ScreenshotApp(): ReactElement {
  const [url, setUrl] = useState<string | undefined>(() => getInjectedData()?.imageUrl)
  const [width, setWidth] = useState(window.innerWidth)
  const [height, setHeight] = useState(window.innerHeight)
  const [display, setDisplay] = useState<Display | undefined>(() => {
    const data = getInjectedData()
    if (data?.displayInfo) {
      try { return JSON.parse(data.displayInfo) } catch (_) {}
    }
    return undefined
  })

  const onCancel = useCallback(() => {
    handleCancel()
  }, [])

  const onOk = useCallback(
    async (blob: Blob | null, bounds: Bounds) => {
      await handleOk(blob, bounds)
    },
    []
  )

  const onSave = useCallback(
    async (blob: Blob | null, bounds: Bounds) => {
      await handleSave(blob, bounds)
    },
    []
  )

  useEffect(() => {
    const onResize = () => {
      setWidth(window.innerWidth)
      setHeight(window.innerHeight)
    }
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])

  useEffect(() => {
    if ((window as any).__TAURI_INTERNALS__) {
      const fetchCapture = async () => {
        try {
          const { invoke } = await import('@tauri-apps/api/core')
          const capture = await invoke<any>('get_screenshot_capture')
          if (capture?.dataUrl) {
            setUrl(capture.dataUrl)
            if (capture.displayInfo) {
              try { setDisplay(JSON.parse(capture.displayInfo)) } catch (_) {}
            }
            return true
          }
        } catch (_) {}
        return false
      }

      const onDataReady = () => {
        const data = getInjectedData()
        if (data?.imageUrl) {
          setUrl(data.imageUrl)
        }
        if (data?.displayInfo) {
          try { setDisplay(JSON.parse(data.displayInfo)) } catch (_) {}
        }
        if (pollRef) clearInterval(pollRef)
      }
      window.addEventListener('screenshot-data-ready', onDataReady)

      let pollRef: any = setInterval(async () => {
        const got = await fetchCapture()
        if (got) clearInterval(pollRef)
      }, 2000)

      fetchCapture().then((got) => { if (got) clearInterval(pollRef) })

      return () => {
        window.removeEventListener('screenshot-data-ready', onDataReady)
        clearInterval(pollRef)
      }
    }

    if (window.runtime) {
      window.runtime.EventsOn('screenshot-data', (dataUrl: string, displayInfo: string) => {
        try {
          setDisplay(JSON.parse(displayInfo))
          setUrl(dataUrl)
        } catch { setUrl(dataUrl) }
      })
      window.runtime.EventsOn('screenshot-cancel', () => {
        setUrl(undefined)
        setDisplay(undefined)
      })
    }
  }, [])

  return (
    <div className="body">
      <Screenshots
        url={url}
        width={width}
        height={height}
        onOk={onOk}
        onCancel={onCancel}
        onSave={onSave}
      />
    </div>
  )
}