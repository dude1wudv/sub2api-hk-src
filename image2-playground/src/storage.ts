import type { ImportedDataState } from '@excalidraw/excalidraw/data/types'

const database = (() => {
  const { promise, resolve, reject } = Promise.withResolvers<IDBDatabase>()
  try {
    const request = indexedDB.open('sub2api-excalidraw', 1)
    request.onupgradeneeded = () => {
      request.result.createObjectStore('scenes')
    }
    request.onsuccess = () => {
      const db = request.result
      db.onversionchange = () => db.close()
      resolve(db)
    }
    request.onerror = () => reject(request.error)
    request.onblocked = () => reject(new Error('画布存储升级被另一个窗口阻止'))
  } catch (error) {
    reject(error)
  }
  return promise
})()

export async function loadScene(): Promise<ImportedDataState | null> {
  const db = await database
  const { promise, resolve, reject } = Promise.withResolvers<ImportedDataState | null>()
  const request = db.transaction('scenes', 'readonly').objectStore('scenes').get('current')
  request.onsuccess = () => {
    try {
      resolve(request.result ? JSON.parse(request.result) as ImportedDataState : null)
    } catch (error) {
      reject(error)
    }
  }
  request.onerror = () => reject(request.error)
  return promise
}

export async function saveScene(scene: string): Promise<void> {
  const db = await database
  const { promise, resolve, reject } = Promise.withResolvers<void>()
  const transaction = db.transaction('scenes', 'readwrite')
  transaction.objectStore('scenes').put(scene, 'current')
  transaction.oncomplete = () => resolve()
  transaction.onabort = () => reject(transaction.error)
  transaction.onerror = () => reject(transaction.error)
  return promise
}
