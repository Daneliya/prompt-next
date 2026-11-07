import { ref, type Ref } from 'vue'
import { PromptService } from '../../bindings/github.com/Daneliya/prompt-next/internal/desktop'

type ImportResult = Awaited<ReturnType<typeof PromptService.ImportChosenBackup>>

export function useBackupImport(
  busy: Ref<boolean>,
  perform: (action: () => Promise<void>) => Promise<void>,
  onImported: (result: ImportResult, replace: boolean) => Promise<void>,
) {
  const pendingImport = ref<(ImportResult & { name: string })>()
  const restoreOpen = ref(false)

  function chooseImport() {
    void perform(async () => {
      const backup = await PromptService.ChooseBackup()
      if (backup.name) pendingImport.value = { name: backup.name, ...backup.summary }
    })
  }

  function closeImport() {
    if (busy.value) return
    pendingImport.value = undefined
    void perform(() => PromptService.ClearChosenBackup())
  }

  function confirmImport(replace = false) {
    if (!pendingImport.value) return
    void perform(async () => {
      const result = await PromptService.ImportChosenBackup(replace)
      pendingImport.value = undefined
      restoreOpen.value = false
      await onImported(result, replace)
    })
  }

  return { pendingImport, restoreOpen, chooseImport, closeImport, confirmImport }
}
