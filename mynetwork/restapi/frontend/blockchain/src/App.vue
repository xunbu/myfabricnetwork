<template>
    <div id="app" class="flex-col flex-grow" style="position: relative;">
        <!-- 顶部导航 -->
        <header class="app-header">
            <div class="container header-content">
                <div class="brand flex-row items-center gap-2" @click="$router.push('/')">
                    <div>
                        <h1>区块链浏览器</h1>
                        <div class="subtitle">企业级监控平台</div>
                    </div>
                </div>

                <nav class="nav-menu">
                    <router-link to="/" custom v-slot="{ navigate, isActive }">
                        <a href="#" @click="navigate" class="nav-link" :class="{ active: isActive }">首页概览</a>
                    </router-link>
                    <router-link to="/nodes" custom v-slot="{ navigate, isActive }">
                        <a href="#" @click="navigate" class="nav-link" :class="{ active: isActive }">节点监控</a>
                    </router-link>
                    <router-link to="/data" custom v-slot="{ navigate, isActive }">
                        <a href="#" @click="navigate" class="nav-link" :class="{ active: isActive }">链上数据</a>
                    </router-link>
                </nav>

                <div class="status-indicator">
                    <div class="text-right hidden" style="display: block;"> <!-- Always show on large, logic in CSS -->
                        <div style="font-size: 12px; font-family: monospace; color: #475569;">{{ refreshStatus }}</div>
                        <div
                            style="font-size: 10px; color: var(--color-success); font-weight: bold; display: flex; align-items: center; justify-content: flex-end;">
                            <span class="status-dot"></span> 运行中
                        </div>
                    </div>
                    <button @click="triggerGlobalRefresh" class="refresh-btn" title="刷新">↻</button>
                </div>
            </div>
        </header>

        <!-- 路由出口 -->
        <main class="container flex-col flex-grow main-content">
            <router-view v-slot="{ Component }">
                <keep-alive :include="['NodesPage']">
                    <component :is="Component" ref="viewComponentRef" />
                </keep-alive>
            </router-view>
        </main>
    </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';

const refreshStatus = ref('--');
const viewComponentRef = ref(null);
let timer = null;

onMounted(() => {
    timer = setInterval(() => {
        refreshStatus.value = new Date().toLocaleTimeString('zh-CN');
    }, 1000);
});

onUnmounted(() => {
    clearInterval(timer);
});

const triggerGlobalRefresh = () => {
    if (viewComponentRef.value && viewComponentRef.value.triggerRefresh) {
        viewComponentRef.value.triggerRefresh();
    }
};
</script>