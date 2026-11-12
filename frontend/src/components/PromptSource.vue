<script setup lang="ts">
import { computed } from 'vue'
import { t } from '../i18n'
import { promptSourceIcon } from '../promptSourceIcons'

const props = defineProps<{ url: string }>()
const emit = defineEmits<{ open: [url: string] }>()
const source = computed(() => {
  try {
    const address = new URL(props.url)
    if ((address.protocol === 'https:' || address.protocol === 'http:') && !address.username && !address.password) {
      return { url: address.href, host: address.hostname, icon: promptSourceIcon(address.hostname) }
    }
  } catch {
    // A legacy import can contain an empty or invalid URL.
  }
  return undefined
})
</script>

<template>
  <button v-if="source" class="prompt-source" type="button" :title="t('打开提示词原链接') + ' · ' + source.host"
    :aria-label="t('打开提示词原链接') + ' · ' + source.host" @click.stop="emit('open', source.url)">
    <img v-if="source.icon" :src="source.icon" alt="">
    <span v-else class="prompt-source-fallback">{{ source.host.replace(/^www\./, '').charAt(0).toUpperCase() }}</span>
    <span class="prompt-source-copy">
      <strong>{{ t('提示词原链接') }}</strong>
      <small>{{ source.host }}</small>
    </span>
    <span aria-hidden="true">↗</span>
  </button>
</template>
