<script setup lang="ts">
import { imageURL } from '../composables/useEditorImages'
import { libraries, videoAsset, type Prompt } from '../prompt'
import AppIcon from './AppIcon.vue'
import VideoCover from './VideoCover.vue'

defineProps<{ item?: Prompt; compact?: boolean }>()
</script>

<template>
  <span class="collection-media" :class="item?.kind" aria-hidden="true">
    <VideoCover v-if="!compact && item?.videos?.length" :key="item.videos[0]" :src="videoAsset(item.number, item.videos[0]!)" />
    <img v-else-if="item?.images?.length" :src="imageURL(item.number + '/' + item.images[0]!, true)" alt="" loading="lazy">
    <AppIcon v-else :name="libraries.find(library => library.kind === item?.kind)?.icon ?? 'folder'" />
  </span>
</template>
