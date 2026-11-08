import { t } from '../i18n'
import { ref, type Ref } from 'vue'

export function imageURL(key: string, thumbnail = false) {
  const route = thumbnail ? '/prompt-thumbnails/' : '/prompt-images/'
  return route + key.split('/').map(encodeURIComponent).join('/')
}

export function readImage(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(new Error(t('读取图片失败')))
    reader.onload = () => resolve(String(reader.result).split(',')[1] ?? '')
    reader.readAsDataURL(file)
  })
}

export type DraftImage = { name: string; url: string; file?: File; sourceNumber?: number }

export function useEditorImages(busy: Ref<boolean>, showImageError: (message: string) => void, otherImages: () => DraftImage[]) {
  const draftImages = ref<DraftImage[]>([])
  const dragFrom = ref<number | null>(null)
  const dragOver = ref<number | null>(null)
  const imageInput = ref<HTMLInputElement>()

  function clearDraftImages() {
    draftImages.value.forEach((item) => { if (item.file) URL.revokeObjectURL(item.url) })
    draftImages.value = []
    endDrag()
  }

  function removeDraftImage(index: number) {
    const [item] = draftImages.value.splice(index, 1)
    if (item?.file) URL.revokeObjectURL(item.url)
  }

  function moveImage(from: number, to: number) {
    if (busy.value || from === to || to < 0 || to >= draftImages.value.length) return
    const [item] = draftImages.value.splice(from, 1)
    if (item) draftImages.value.splice(to, 0, item)
  }

  function startDrag(event: DragEvent, index: number) {
    if (busy.value || !event.dataTransfer) {
      event.preventDefault()
      return
    }
    dragFrom.value = index
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', String(index))
  }

  function overImage(event: DragEvent, index: number) {
    if (dragFrom.value === null || !event.dataTransfer) return
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
    dragOver.value = index
  }

  function dropImage(event: DragEvent, index: number) {
    if (dragFrom.value === null) return
    event.preventDefault()
    moveImage(dragFrom.value, index)
    endDrag()
  }

  function endDrag() {
    dragFrom.value = null
    dragOver.value = null
  }

  function addImages(event: Event) {
    const input = event.target as HTMLInputElement
    const files = Array.from(input.files ?? [])
    input.value = ''
    if (files.some((file) => file.size > 12 * 1024 * 1024)) {
      showImageError(t('每张图片须在 12 MB 以内'))
      return
    }
    const all = [...draftImages.value, ...otherImages()]
    const batchSize = all.reduce((sum, item) => sum + (item.file?.size ?? 0), 0) +
      files.reduce((sum, file) => sum + file.size, 0)
    if (all.length + files.length > 48 || batchSize > 80 * 1024 * 1024) {
      showImageError(t('每条最多 48 张图片，每次保存的新图片合计须在 80 MB 以内'))
      return
    }
    draftImages.value.push(...files.map((file) => ({ name: '', file, url: URL.createObjectURL(file) })))
  }

  return { draftImages, dragFrom, dragOver, imageInput,
    clearDraftImages, removeDraftImage, moveImage, startDrag, overImage, dropImage, endDrag, addImages }
}
