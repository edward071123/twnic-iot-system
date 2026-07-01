<template>
  <WardFloorMonitor :show-thermal="showThermal" :engineer-mode="engineerMode" />
</template>

<script setup>
import WardFloorMonitor from '../components/WardFloorMonitor.vue'

defineProps({
  showThermal: {
    type: Boolean,
    default: true,
  },
  engineerMode: {
    type: Boolean,
    default: false,
  },
})
</script>
