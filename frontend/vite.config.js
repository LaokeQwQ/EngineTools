import {createServer} from 'node:net'
import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
const host = '127.0.0.1'

function findDevPort() {
  return new Promise((resolve, reject) => {
    const socket = createServer()
    socket.once('error', reject)
    socket.listen(0, host, () => {
      const {port} = socket.address()
      socket.close(error => error ? reject(error) : resolve(port))
    })
  })
}

export default defineConfig(async ({command, isPreview}) => ({
  plugins: [vue()],
  // Vite treats port 0 as its default port. Resolve a concrete port first.
  server: command === 'serve' && !isPreview ? {host, port: await findDevPort()} : undefined,
}))
