<script setup>
defineProps({
    msgs: { type: Object, default: () => ({}) },
    loading: { type: Boolean, default: false },
})
</script>
<template>
    <div class="content" v-show="true">
        <div class="info-card">
            <div class="info-row">
                <span class="info-label">Metadata Editor</span>
            </div>
        </div>
    </div>
</template>
