<script setup lang="ts">
import { ref } from 'vue'

const props = withDefaults(defineProps<{ title: string; initiallyOpen?: boolean }>(), {
  initiallyOpen: true,
})
const expanded = ref(props.initiallyOpen)
</script>

<template>
  <section class="prompt-fold">
    <header class="prompt-fold-header">
      <button class="prompt-fold-toggle" type="button" :aria-expanded="expanded" @click="expanded = !expanded">
        <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m6 4 4 4-4 4" /></svg>
        <strong>{{ title }}</strong>
      </button>
      <div v-if="$slots.actions" class="prompt-fold-actions"><slot name="actions" /></div>
    </header>
    <div v-show="expanded" class="prompt-fold-content"><slot /></div>
  </section>
</template>
