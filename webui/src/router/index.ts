import { createRouter, createWebHistory } from 'vue-router'
import { isLoggedIn, getUser } from '@/lib/auth'
import LoginView from '@/views/LoginView.vue'
import BugsView from '@/views/BugsView.vue'
import ReportBugView from '@/views/ReportBugView.vue'
import BugHistoryView from '@/views/BugHistoryView.vue'
import BugDetailView from '@/views/BugDetailView.vue'
import NotificationsView from '@/views/NotificationsView.vue'
import ProfileView from '@/views/ProfileView.vue'
import SettingsView from '@/views/SettingsView.vue'
import StatsView from '@/views/StatsView.vue'

const router = createRouter({
  history: createWebHistory('/bugs/'),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
    { path: '/', redirect: '/bugs' },
    { path: '/stats', name: 'stats', component: StatsView },
    { path: '/bugs', name: 'bugs', component: BugsView },
    { path: '/bugs/:id', name: 'bug-detail', component: BugDetailView },
    { path: '/bugs/:id/history', name: 'bug-history', component: BugHistoryView },
    {
      path: '/report-bug',
      name: 'report-bug',
      component: ReportBugView,
      meta: { roles: ['tester', 'developer', 'manager', 'admin'] },
    },
    { path: '/notifications', name: 'notifications', component: NotificationsView },
    { path: '/profile', name: 'profile', component: ProfileView },
    { path: '/profile/:username', name: 'user-profile', component: ProfileView },
    { path: '/settings', name: 'settings', component: SettingsView },
  ],
})

router.beforeEach((to) => {
  if (to.meta.public) {
    if (isLoggedIn() && to.name === 'login') return { name: 'bugs' }
    return true
  }
  if (!isLoggedIn()) return { name: 'login' }
  const allowed = to.meta.roles as string[] | undefined
  if (allowed?.length) {
    const role = getUser()?.role
    if (!role || !allowed.includes(role)) return { name: 'bugs' }
  }
  return true
})

export default router
