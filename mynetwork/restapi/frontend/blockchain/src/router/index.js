import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import NodesView from '../views/NodesView.vue'
import DataView from '../views/DataView.vue'
import BlockDetailView from '../views/BlockDetailView.vue'

const routes = [
    { path: '/', component: HomeView },
    { path: '/nodes', component: NodesView },
    { path: '/data', component: DataView },
    { path: '/block/:blockId', component: BlockDetailView }
]

const router = createRouter({
    history: createWebHashHistory(),
    routes
})

export default router