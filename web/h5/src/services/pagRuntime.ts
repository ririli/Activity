import { PAGInit } from 'libpag'

let pagRuntime: ReturnType<typeof PAGInit> | null = null

export function getPagRuntime() {
  if (!pagRuntime) {
    pagRuntime = PAGInit({
      locateFile: () => '/libpag.wasm',
    })
  }
  return pagRuntime
}
