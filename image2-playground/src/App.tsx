import { useEffect, useRef, useState } from 'react'
import { Excalidraw, MainMenu, serializeAsJSON } from '@excalidraw/excalidraw'
import type { AppState, BinaryFiles } from '@excalidraw/excalidraw/types'
import type { ExcalidrawElement } from '@excalidraw/excalidraw/element/types'
import { loadScene, saveScene } from './storage'
import '@excalidraw/excalidraw/index.css'

export default function App() {
  const [storageError, setStorageError] = useState(false)
  const storageLoaded = useRef(false)
  const latestScene = useRef<string | null>(null)
  const saveTimer = useRef<number | undefined>(undefined)
  const [initialData] = useState(() => loadScene().then((scene) => {
    storageLoaded.current = true
    return scene
  }).catch(() => {
    setStorageError(true)
    return null
  }))

  useEffect(() => {
    const flush = () => {
      clearTimeout(saveTimer.current)
      if (latestScene.current !== null) {
        void saveScene(latestScene.current).then(() => {
          setStorageError(false)
        }).catch(() => {
          setStorageError(true)
        })
      }
    }
    const onVisibilityChange = () => {
      if (document.visibilityState === 'hidden') flush()
    }
    window.addEventListener('pagehide', flush)
    document.addEventListener('visibilitychange', onVisibilityChange)
    return () => {
      flush()
      window.removeEventListener('pagehide', flush)
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  }, [])

  function persistScene(elements: readonly ExcalidrawElement[], appState: AppState, files: BinaryFiles) {
    if (!storageLoaded.current) return
    const scene = JSON.parse(serializeAsJSON(elements, appState, files, 'local'))
    scene.appState = {
      ...scene.appState,
      scrollX: appState.scrollX,
      scrollY: appState.scrollY,
      zoom: appState.zoom,
      theme: appState.theme,
    }
    latestScene.current = JSON.stringify(scene)
    clearTimeout(saveTimer.current)
    saveTimer.current = window.setTimeout(() => {
      void saveScene(latestScene.current!).then(() => {
        setStorageError(false)
      }).catch(() => {
        setStorageError(true)
      })
    }, 300)
  }

  return (
    <main className="canvas-shell">
      {storageError && (
        <div className="storage-warning" role="alert">
          自动保存不可用，请通过菜单将画布保存为文件，避免关闭页面后丢失内容。
        </div>
      )}
      <Excalidraw initialData={initialData} onChange={persistScene} langCode="zh-CN">
        <MainMenu>
          <MainMenu.DefaultItems.LoadScene />
          <MainMenu.DefaultItems.SaveToActiveFile />
          <MainMenu.DefaultItems.Export />
          <MainMenu.DefaultItems.SaveAsImage />
          <MainMenu.DefaultItems.SearchMenu />
          <MainMenu.DefaultItems.ClearCanvas />
          <MainMenu.Separator />
          <MainMenu.DefaultItems.ToggleTheme />
          <MainMenu.DefaultItems.ChangeCanvasBackground />
          <MainMenu.Separator />
          <MainMenu.ItemLink href="/dashboard">返回 Sub2API</MainMenu.ItemLink>
        </MainMenu>
      </Excalidraw>
    </main>
  )
}
