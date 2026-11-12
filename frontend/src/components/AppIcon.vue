<script setup lang="ts">
defineProps<{ name: string }>()
</script>

<template>
  <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
    <template v-if="name === 'home'">
      <path d="m2.5 11 9.5-8 9.5 8M5 9.5V20h5v-6h4v6h5V9.5" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/>
    </template>
    <template v-else-if="name === 'art'">
      <rect x="3" y="4" width="18" height="16" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <path d="m4 17 5-5 4 4 3-3 4 4" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/>
      <circle cx="15.5" cy="9" r="1.4" fill="currentColor"/>
    </template>
    <template v-else-if="name === 'fish'">
      <path d="M6 12C9 7 13 6 16.5 8 19 9 21 11 21 12s-2 3-4.5 4C13 18 9 17 6 12Zm0 0-3.5-3v6L6 12Z" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/>
      <circle cx="16.5" cy="11" r="1" fill="currentColor"/>
    </template>
    <path v-else-if="name === 'star' || name === 'star-outline'"
      d="m12 3 2.8 5.7 6.3.9-4.6 4.5 1.1 6.3-5.6-3-5.6 3 1.1-6.3L3 9.6l6.2-.9L12 3Z"
      :fill="name === 'star' ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round" />
    <template v-else-if="name === 'persona'">
      <circle cx="12" cy="8" r="3.5" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <path d="M4 20v-1a8 5 0 0 1 16 0v1" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/>
    </template>
    <template v-else-if="name === 'disk'">
      <ellipse cx="12" cy="6" rx="8" ry="3" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <path d="M4 6v11c0 4 16 4 16 0V6M4 12c0 4 16 4 16 0" fill="none" stroke="currentColor" stroke-width="1.7"/>
    </template>
    <template v-else-if="name === 'video'">
      <rect x="3" y="5" width="18" height="14" rx="3" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <path d="m10 9 5 3-5 3z" fill="currentColor"/>
    </template>
    <template v-else-if="name === 'trash'">
      <path d="M4 6h16m-14 0 1 14h10l1-14M9 6V3h6v3m-5 4v6m4-6v6" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
    </template>
    <template v-else-if="name === 'text'">
      <path d="M6 3.5h8l4 4V20.5H6zM14 3.5v4h4M9 12h6m-6 4h6" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
    </template>
    <template v-else-if="name === 'waterfall'">
      <path d="M3 4h7v10H3zM14 4h7v6h-7zM3 17h7v3H3zM14 13h7v7h-7z" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
    </template>
    <template v-else-if="name === 'cards'">
      <rect x="3" y="5" width="8" height="14" rx="2" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <rect x="14" y="5" width="7" height="14" rx="2" fill="none" stroke="currentColor" stroke-width="1.7"/>
    </template>
    <template v-else-if="name === 'folder'">
      <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/>
    </template>
    <template v-else-if="name === 'tag'">
      <path d="M3 4h8l9 9a2 2 0 0 1 0 2.8l-4.2 4.2a2 2 0 0 1-2.8 0l-10-10Z" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/>
      <circle cx="7.5" cy="8.5" r="1.3" fill="currentColor"/>
    </template>
    <template v-else-if="name === 'list'">
      <path d="M8 6h13M8 12h13M8 18h13M3 6h1M3 12h1M3 18h1" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
    </template>
    <template v-else-if="name === 'chevron-left' || name === 'chevron-right'">
      <path :d="name === 'chevron-left' ? 'm14.5 6.5-5 5.5 5 5.5' : 'm9.5 6.5 5 5.5-5 5.5'"
        fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    </template>
    <template v-else-if="name === 'search'">
      <circle cx="10.5" cy="10.5" r="5.5" fill="none" stroke="currentColor" stroke-width="1.8"/>
      <path d="m15 15 4 4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
    </template>
    <template v-else-if="name === 'import' || name === 'backup'">
      <path d="M4 15v4h16v-4" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
      <path :d="name === 'import' ? 'M12 4v10m-4-4 4 4 4-4' : 'M12 14V4M8 8l4-4 4 4'"
        fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
    </template>
    <template v-else-if="name === 'window-restore'">
      <path d="M5 8h11v11H5zM8 8V5h11v11h-3" fill="none" stroke="currentColor" stroke-width="1.5"/>
    </template>
    <template v-else-if="name === 'window-minimise' || name === 'window-maximise' || name === 'window-close'">
      <path :d="name === 'window-minimise' ? 'M5 12h14' : name === 'window-maximise' ? 'M5 5h14v14H5z' : 'm5 5 14 14M19 5 5 19'"
        fill="none" stroke="currentColor" stroke-width="1.5"/>
    </template>
    <template v-else-if="name === 'keyboard'">
      <rect x="3" y="6" width="18" height="12" rx="2" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <path d="M7 10h.1M12 10h.1M17 10h.1M7 14h.1M10 14h7" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/>
    </template>
    <template v-else-if="name === 'settings'">
      <circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <path d="M10 3h4l.6 2.5 2 .9 2.3-1 2 3.5-1.8 1.7v2.8l1.8 1.7-2 3.5-2.3-1-2 .9L14 21h-4l-.6-2.5-2-.9-2.3 1-2-3.5 1.8-1.7v-2.8L3.1 8.9l2-3.5 2.3 1 2-.9z" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"/>
    </template>
  </svg>
</template>
