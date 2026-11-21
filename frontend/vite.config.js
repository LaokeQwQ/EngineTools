import {createServer} from 'node:net'
import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
const host = '127.0.0.1'

function findDevPort() {
  // Vite reloads this config when the file changes but keeps the process, so
  // remember the chosen port instead of switching ports on every reload.
  const saved = Number(process.env.WAILS_DEV_PORT)
  if (saved > 0) return Promise.resolve(saved)
  return new Promise((resolve, reject) => {
    const socket = createServer()
    socket.once('error', reject)
    socket.listen(0, host, () => {
      const {port} = socket.address()
      socket.close(error => {
        if (error) return reject(error)
        process.env.WAILS_DEV_PORT = String(port)
        resolve(port)
      })
    })
  })
}

export default defineConfig(async ({command, isPreview}) => ({
  plugins: [vue()],
  // Vite treats port 0 as its default port. Resolve a concrete port first.
  server: command === 'serve' && !isPreview ? {host, port: await findDevPort()} : undefined,
}))
