import { defineConfig } from "vite"
import { devtools } from "@tanstack/devtools-vite"
import { tanstackStart } from "@tanstack/react-start/plugin/vite"
import viteReact from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

const dashboardHosts = [
  process.env.SHIP_DASHBOARD_HOST ?? "k8s.gron-studio.com",
]

const config = defineConfig({
  resolve: { tsconfigPaths: true },
  server: {
    allowedHosts: dashboardHosts,
  },
  preview: {
    allowedHosts: dashboardHosts,
  },
  plugins: [devtools(), tailwindcss(), tanstackStart(), viteReact()],
})

export default config
