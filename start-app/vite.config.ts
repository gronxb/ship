import { defineConfig } from "vite"
import { devtools } from "@tanstack/devtools-vite"
import { tanstackStart } from "@tanstack/react-start/plugin/vite"
import viteReact from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

export const dashboardAllowedHosts = () => [
  process.env.SHIP_DASHBOARD_HOST ??
    (process.env.SHIP_DOMAIN ? `k8s.${process.env.SHIP_DOMAIN}` : undefined) ??
    "k8s.example.com",
]

const dashboardHosts = dashboardAllowedHosts()

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
