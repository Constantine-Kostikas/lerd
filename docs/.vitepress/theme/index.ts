// The site sets its own type, so skip the default theme's Inter preload.
import DefaultTheme from 'vitepress/theme-without-fonts'
import Layout from './Layout.vue'
import './style.css'
import './landing.css'

// /changelog keeps only recent releases, so an older release's #anchor moves on to the archive.
function forwardOldRelease() {
  const id = decodeURIComponent(location.hash.slice(1))
  if (/^\/changelog(\.html)?$/.test(location.pathname) && id && !document.getElementById(id)) {
    location.replace(`/changelog-archive${location.hash}`)
  }
}

export default {
  extends: DefaultTheme,
  Layout,
  enhanceApp({ router }) {
    if (typeof window === 'undefined') return
    forwardOldRelease()
    router.onAfterRouteChange = forwardOldRelease
  },
}
