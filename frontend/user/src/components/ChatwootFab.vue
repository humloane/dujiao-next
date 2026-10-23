<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { MessageCircle } from 'lucide-vue-next'
import { openChatwoot } from '../utils/chatwoot'

const { t } = useI18n()
const route = useRoute()

const visible = computed(() => {
  if (route.name === 'landing-page') return false
  return !route.path.startsWith('/checkout') && !route.path.startsWith('/pay')
})

const handleClick = () => {
  openChatwoot()
}
</script>

<template>
  <button
    v-if="visible"
    type="button"
    class="chatwoot-fab"
    :aria-label="t('chatwoot.online')"
    @click="handleClick"
  >
    <span class="chatwoot-fab__icon" aria-hidden="true">
      <MessageCircle :size="22" :stroke-width="2.2" />
    </span>
    <span class="chatwoot-fab__label">{{ t('chatwoot.online') }}</span>
  </button>
</template>

<style scoped>
.chatwoot-fab {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 70;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 0 20px;
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: 999px;
  background: linear-gradient(135deg, #1f6feb, #0b4fc4);
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
  cursor: pointer;
  box-shadow: 0 14px 34px rgba(15, 76, 190, 0.35), 0 3px 10px rgba(0, 0, 0, 0.18);
  animation: chatwoot-fab-breathe 3.2s ease-in-out infinite;
  transition: transform 180ms ease, box-shadow 180ms ease;
}

.chatwoot-fab:hover {
  transform: translateY(-2px);
  box-shadow: 0 18px 40px rgba(15, 76, 190, 0.42), 0 4px 12px rgba(0, 0, 0, 0.2);
}

.chatwoot-fab:active {
  transform: translateY(0) scale(0.98);
}

.chatwoot-fab__icon {
  display: inline-grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.18);
}

@keyframes chatwoot-fab-breathe {
  0%, 100% { box-shadow: 0 14px 34px rgba(15, 76, 190, 0.35), 0 3px 10px rgba(0, 0, 0, 0.18); }
  50% { box-shadow: 0 18px 42px rgba(15, 76, 190, 0.5), 0 3px 12px rgba(0, 0, 0, 0.2); }
}

@media (max-width: 640px) {
  .chatwoot-fab {
    right: 16px;
    bottom: calc(env(safe-area-inset-bottom, 0px) + 84px);
    min-height: 46px;
    padding: 0 16px;
    font-size: 14px;
  }

  .chatwoot-fab__icon {
    width: 26px;
    height: 26px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chatwoot-fab {
    animation: none;
  }
}
</style>
