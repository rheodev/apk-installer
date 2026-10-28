<script lang="ts" setup>
import {computed, nextTick, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {CancelApk, GetAdbInfo, InstallApk, ListDevices, SelectApk} from '../wailsjs/go/main/App'
import {main} from '../wailsjs/go/models'
import {EventsOff, EventsOn, OnFileDrop, OnFileDropOff} from '../wailsjs/runtime/runtime'

type AdbInfo = main.AdbInfo
type Device = main.Device
type InstallLog = {
  level: string
  message: string
  time: string
}

type ThemeMode = 'light' | 'dark'

const THEME_KEY = 'apk-install-theme'

const adbInfo = ref<AdbInfo>({
  available: false,
  path: '',
  source: '',
  version: '',
  message: '检查中',
})
const devices = ref<Device[]>([])
const selectedSerial = ref('')
const apkPath = ref('')
const logs = ref<InstallLog[]>([])
const statusText = ref('准备就绪')
const loadingDevices = ref(false)
const installing = ref(false)
const cancelling = ref(false)
const dragging = ref(false)
const deviceMenuOpen = ref(false)
const logPanel = ref<HTMLElement | null>(null)
const theme = ref<ThemeMode>(resolveTheme())

const onlineDevices = computed(() => devices.value.filter(device => device.state === 'device'))
const selectedDevice = computed(() => devices.value.find(device => device.serial === selectedSerial.value))
const themeLabel = computed(() => theme.value === 'dark' ? '深色' : '浅色')
const fileName = computed(() => apkPath.value.split(/[\\/]/).pop() || '')
const canInstall = computed(() => {
  return adbInfo.value.available && selectedDevice.value?.state === 'device' && apkPath.value !== '' && !installing.value
})
const adbVersionShort = computed(() => {
  const match = adbInfo.value.version.match(/(\d+(\.\d+)+)/)
  return match ? match[1] : ''
})
const adbLabel = computed(() => {
  if (!adbInfo.value.available) {
    return adbInfo.value.message
  }
  return adbVersionShort.value ? `adb ${adbVersionShort.value}` : 'adb'
})
const showLogDock = computed(() => logs.value.length > 0)

function resolveTheme(): ThemeMode {
  if (typeof window === 'undefined') {
    return 'dark'
  }

  const saved = window.localStorage.getItem(THEME_KEY)
  if (saved === 'light' || saved === 'dark') {
    return saved
  }

  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

function applyTheme(mode: ThemeMode) {
  if (typeof document === 'undefined') {
    return
  }

  document.documentElement.dataset.theme = mode
  document.documentElement.style.colorScheme = mode
  window.localStorage.setItem(THEME_KEY, mode)
}

function deviceName(device: Device) {
  return device.model || device.device || device.product || device.serial
}

function addLog(level: string, message: string) {
  logs.value.push({
    level,
    message,
    time: new Date().toLocaleTimeString('zh-CN', {hour12: false}),
  })
  void nextTick(() => {
    if (logPanel.value) {
      logPanel.value.scrollTop = logPanel.value.scrollHeight
    }
  })
}

async function refreshAdb() {
  adbInfo.value = await GetAdbInfo()
  if (!adbInfo.value.available) {
    statusText.value = adbInfo.value.message
  }
}

async function refreshDevices() {
  loadingDevices.value = true
  try {
    devices.value = await ListDevices()
    const stillExists = devices.value.some(device => device.serial === selectedSerial.value)
    if (!stillExists) {
      selectedSerial.value = onlineDevices.value[0]?.serial || devices.value[0]?.serial || ''
    }
    statusText.value = devices.value.length > 0 ? `发现 ${devices.value.length} 台设备` : '未发现设备'
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    devices.value = []
    selectedSerial.value = ''
    statusText.value = message
    addLog('error', message)
  } finally {
    loadingDevices.value = false
  }
}

async function chooseApk() {
  try {
    const path = await SelectApk()
    if (path) {
      apkPath.value = path
      statusText.value = '安装文件已选择'
    }
  } catch (error) {
    addLog('error', error instanceof Error ? error.message : String(error))
  }
}

async function install() {
  if (!canInstall.value) {
    return
  }

  installing.value = true
  logs.value = []
  statusText.value = '安装中'

  try {
    const result = await InstallApk({
      deviceSerial: selectedSerial.value,
      apkPath: apkPath.value,
    })
    if (result.cancelled) {
      statusText.value = '已取消'
    } else if (result.success) {
      statusText.value = '安装成功'
    } else {
      statusText.value = '安装失败'
      if (result.error) {
        addLog('error', result.error)
      }
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    statusText.value = '安装失败'
    addLog('error', message)
  } finally {
    installing.value = false
  }
}

async function cancelInstall() {
  cancelling.value = true
  try {
    await CancelApk()
  } catch (error) {
    addLog('error', error instanceof Error ? error.message : String(error))
  } finally {
    cancelling.value = false
  }
}

async function onDrop(path: string) {
  if (!/\.(apk|xapk|apkm|apks)$/i.test(path)) {
    addLog('error', `不支持的文件类型: ${path}`)
    statusText.value = '仅支持 .apk / .xapk / .apkm / .apks'
    return
  }

  apkPath.value = path

  const online = onlineDevices.value
  if (online.length === 0) {
    statusText.value = '未发现可用设备,请连接设备后重试'
    return
  }
  if (online.length > 1) {
    statusText.value = '已选择安装包,请在设备下拉中选择目标设备'
    deviceMenuOpen.value = true
    return
  }
  if (installing.value) {
    return
  }
  selectedSerial.value = online[0].serial
  await install()
}

function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
}

watch(theme, mode => applyTheme(mode), {immediate: true})

onMounted(async () => {
  EventsOn('adb-install-log', (entry: InstallLog) => {
    logs.value.push(entry)
    void nextTick(() => {
      if (logPanel.value) {
        logPanel.value.scrollTop = logPanel.value.scrollHeight
      }
    })
  })

  OnFileDrop((_x, _y, paths) => {
    dragging.value = false
    const path = paths[0]
    if (path) {
      void onDrop(path)
    }
  }, false)

  await refreshAdb()
  if (adbInfo.value.available) {
    await refreshDevices()
  }
})

onBeforeUnmount(() => {
  EventsOff('adb-install-log')
  OnFileDropOff()
})
</script>

<template>
  <main class="shell" :class="{dragging}" @click="deviceMenuOpen = false" @keydown.esc="deviceMenuOpen = false" @dragenter.prevent="dragging = true" @dragover.prevent @dragleave="dragging = false" @drop.prevent="dragging = false">
    <section class="titlebar">
      <div class="brand">
        <span class="brand-mark" aria-hidden="true"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3 8 4.5v9L12 21l-8-4.5v-9L12 3Z" /><path d="m4 7.5 8 4.5 8-4.5M12 12v9M8 5.3l8 4.5" /></svg></span>
        <strong>APK 安装器</strong>
        <span class="adb-version" :class="{ok: adbInfo.available}" :title="adbInfo.available ? adbInfo.version || 'adb 可用' : adbInfo.message">
          {{ adbLabel }}
        </span>
      </div>
      <div class="titlebar-actions">
        <div class="status" role="status" :title="statusText" :class="{ready: adbInfo.available && onlineDevices.length > 0}">
          <span class="status-dot"></span>
          <span>{{ statusText }}</span>
        </div>
        <button class="theme-toggle" type="button" :aria-label="`切换到${theme === 'dark' ? '浅色' : '深色'}模式`" @click="toggleTheme">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path v-if="theme === 'dark'" d="M20.5 13A8.5 8.5 0 0 1 11 3.5 8.5 8.5 0 1 0 20.5 13Z" />
            <g v-else><circle cx="12" cy="12" r="4" /><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5" /></g>
          </svg>
          <span>{{ themeLabel }}</span>
        </button>
      </div>
    </section>

    <section class="stage">
      <div v-if="apkPath" class="file-card">
        <div class="file-icon" aria-hidden="true"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M14 3H6v18h12V7l-4-4Z" /><path d="M14 3v5h4M9 13h6m-6 4h4" /></svg></div>
        <div class="file-meta">
          <strong class="file-name" :title="apkPath">{{ fileName }}</strong>
          <small class="file-path" :title="apkPath">{{ apkPath }}</small>
        </div>
        <button class="ghost-button" type="button" @click="chooseApk">更换</button>
      </div>

      <div class="drop-zone" :class="{armed: !apkPath}" role="button" tabindex="0" aria-label="选择安装包" @click="chooseApk" @keydown.enter.prevent="chooseApk" @keydown.space.prevent="chooseApk">
        <div class="drop-glyph" aria-hidden="true"><svg viewBox="0 0 32 32" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21V5m-6 6 6-6 6 6M6 20v6a1 1 0 0 0 1 1h18a1 1 0 0 0 1-1v-6" /></svg></div>
        <p class="drop-title">拖入安装包</p>
        <p class="drop-hint">或点击选择 · 支持 .apk / .xapk / .apkm / .apks</p>
      </div>

      <div class="action-row">
        <div class="device-select" @click.stop :class="{open: deviceMenuOpen, empty: onlineDevices.length === 0}">
          <button
            class="device-select-button"
            type="button"
            :aria-expanded="deviceMenuOpen"
            aria-haspopup="listbox"
            :disabled="devices.length === 0"
            @click="deviceMenuOpen = !deviceMenuOpen"
          >
            <span class="device-dot" :class="{online: selectedDevice?.state === 'device'}"></span>
            <span class="device-select-label">
              <template v-if="selectedDevice">
                <strong>{{ deviceName(selectedDevice) }}</strong>
                <small>{{ selectedDevice.serial }}</small>
              </template>
              <template v-else>{{ devices.length === 0 ? '未发现设备' : '选择设备' }}</template>
            </span>
            <svg class="icon chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg>
          </button>
          <button class="ghost-button square" type="button" :disabled="loadingDevices || !adbInfo.available" :aria-label="loadingDevices ? '刷新中' : '刷新设备列表'" @click="refreshDevices">
            <svg class="icon" :class="{spinning: loadingDevices}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 7v5h-5M4 17v-5h5" /><path d="M6.1 7a7 7 0 0 1 11.5-1L20 9M4 15l2.4 3A7 7 0 0 0 17.9 17" /></svg>
          </button>
          <div v-if="deviceMenuOpen" class="device-menu" role="listbox" aria-label="设备列表">
            <button
              v-for="device in devices"
              :key="device.serial"
              class="device-option"
              :class="{active: selectedSerial === device.serial, blocked: device.state !== 'device'}"
              type="button"
              role="option"
              :aria-selected="selectedSerial === device.serial"
              @click="selectedSerial = device.serial; deviceMenuOpen = false"
            >
              <span class="device-dot" :class="{online: device.state === 'device'}"></span>
              <span class="device-select-label">
                <strong>{{ deviceName(device) }}</strong>
                <small>{{ device.serial }}</small>
              </span>
              <span class="device-state">{{ device.state === 'device' ? '在线' : device.state }}</span>
            </button>
          </div>
        </div>

        <button class="install-button" type="button" :disabled="!canInstall" @click="install">
          {{ installing ? '安装中···' : '安装' }}
        </button>
        <button v-if="installing" class="cancel-button" type="button" :disabled="cancelling" @click="cancelInstall">
          {{ cancelling ? '取消中···' : '取消' }}
        </button>
      </div>
    </section>

    <transition name="dock">
      <section v-if="showLogDock" class="log-dock" aria-label="安装日志">
        <header class="dock-head">
          <span class="dock-title">日志</span>
          <span class="dock-count">{{ logs.length }} 条</span>
          <button class="dock-close" type="button" aria-label="清空日志" @click="logs = []"><svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" aria-hidden="true"><path d="m7 7 10 10M7 17 17 7" /></svg></button>
        </header>
        <div ref="logPanel" class="dock-body">
          <div v-for="(log, index) in logs" :key="index" class="log-line" :class="log.level">
            <span>{{ log.time }}</span>
            <code>{{ log.message }}</code>
          </div>
        </div>
      </section>
    </transition>

    <div v-if="dragging" class="drag-veil" aria-hidden="true">
      <p>松手开始安装</p>
    </div>
  </main>
</template>
