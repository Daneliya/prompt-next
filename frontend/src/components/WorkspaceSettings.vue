<script lang="ts">
const settingsSections = [
  { key: 'appearance', label: '外观', icon: 'settings' },
  { key: 'shortcuts', label: '快捷键', icon: 'keyboard' },
  { key: 'models', label: 'AI 模型', icon: 'art' },
  { key: 'categories', label: '分类', icon: 'folder' },
  { key: 'tags', label: '标签', icon: 'tag' },
  { key: 'data', label: '数据管理', icon: 'disk' },
  { key: 'about', label: '关于', icon: 'persona' },
] as const

export type SettingsSection = (typeof settingsSections)[number]['key']
</script>

<script setup lang="ts">
import { ref } from 'vue'
import { NButton, NModal } from 'naive-ui'
import { t } from '../i18n'
import AppIcon from './AppIcon.vue'

defineProps<{ isDark: boolean; isMac: boolean }>()
const show = defineModel<boolean>('show', { required: true })
const section = defineModel<SettingsSection>('section', { required: true })
const heading = ref<HTMLElement>()
</script>

<template>
  <NModal v-model:show="show" preset="card" class="settings-modal" :auto-focus="false" @after-enter="heading?.focus()"
    :class="{ 'is-dark': isDark }" :bordered="false" :mask-closable="false">
    <template #header><span ref="heading" class="settings-heading" tabindex="-1">{{ t('偏好设置') }}</span></template>
    <div class="settings-layout">
      <nav class="settings-nav" :aria-label="t('设置分类')">
        <button v-for="item in settingsSections" :key="item.key" type="button" :aria-pressed="section === item.key" @click="section = item.key">
          <AppIcon :name="item.icon" class="ui-icon" />{{ t(item.label) }}
        </button>
      </nav>
      <section class="settings-content" :aria-label="t(settingsSections.find(item => item.key === section)?.label)">
        <template v-if="section === 'shortcuts'">
          <div class="setting-field">
            <h3>{{ t('工作区快捷键') }}</h3>
            <dl class="shortcut-list">
              <div><dt>{{ t('新建提示词') }}</dt><dd><kbd>{{ isMac ? '⌘' : 'Ctrl' }}</kbd><span>+</span><kbd>N</kbd></dd></div>
              <div><dt>{{ t('聚焦搜索框') }}</dt><dd><kbd>{{ isMac ? '⌘' : 'Ctrl' }}</kbd><span>+</span><kbd>F</kbd></dd></div>
            </dl>
          </div>
          <div class="setting-field">
            <h3>{{ t('图片预览') }}</h3>
            <p class="setting-hint">{{ t('焦点在图片或切换按钮上时，可以用方向键切换。') }}</p>
            <dl class="shortcut-list">
              <div><dt>{{ t('上一张') }}</dt><dd><kbd>←</kbd></dd></div>
              <div><dt>{{ t('下一张') }}</dt><dd><kbd>→</kbd></dd></div>
              <div><dt>{{ t('关闭预览') }}</dt><dd><kbd>Esc</kbd></dd></div>
            </dl>
          </div>
        </template>
        <slot v-else />
      </section>
    </div>
    <template #footer><div class="settings-footer"><NButton type="primary" @click="show = false">{{ t('完成') }}</NButton></div></template>
  </NModal>
</template>
