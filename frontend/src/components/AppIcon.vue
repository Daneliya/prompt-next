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
    <template v-else-if="name === 'persona'">
      <circle cx="12" cy="8" r="3.5" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <path d="M4 20v-1a8 5 0 0 1 16 0v1" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/>
    </template>
    <template v-else-if="name === 'disk'">
      <ellipse cx="12" cy="6" rx="8" ry="3" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <path d="M4 6v11c0 4 16 4 16 0V6M4 12c0 4 16 4 16 0" fill="none" stroke="currentColor" stroke-width="1.7"/>
    </template>
    <template v-else-if="name === 'cards'">
      <rect x="3" y="5" width="8" height="14" rx="2" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <rect x="14" y="5" width="7" height="14" rx="2" fill="none" stroke="currentColor" stroke-width="1.7"/>
    </template>
    <template v-else-if="name === 'list'">
      <path d="M8 6h13M8 12h13M8 18h13M3 6h1M3 12h1M3 18h1" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
    </template>
    <template v-else-if="name === 'search'">
      <circle cx="10.5" cy="10.5" r="5.5" fill="none" stroke="currentColor" stroke-width="1.8"/>
      <path d="m15 15 4 4" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
    </template>
    <template v-else-if="name === 'settings'">
      <circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.7"/>
      <path d="M10 3h4l.6 2.5 2 .9 2.3-1 2 3.5-1.8 1.7v2.8l1.8 1.7-2 3.5-2.3-1-2 .9L14 21h-4l-.6-2.5-2-.9-2.3 1-2-3.5 1.8-1.7v-2.8L3.1 8.9l2-3.5 2.3 1 2-.9z" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"/>
    </template>
  </svg>
</template>
