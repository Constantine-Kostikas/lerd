// The site sets its own type, so skip the default theme's Inter preload.
import DefaultTheme from 'vitepress/theme-without-fonts'
import Layout from './Layout.vue'
import './style.css'
import './landing.css'

export default {
  extends: DefaultTheme,
  Layout,
}
