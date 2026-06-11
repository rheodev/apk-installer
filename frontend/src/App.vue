<script lang="ts" setup>
import {computed, nextTick, onBeforeUnmount, onMounted, ref} from 'vue'
import {GetAdbInfo, InstallApk, ListDevices, SelectApk} from '../wailsjs/go/main/App'
import {EventsOff, EventsOn} from '../wailsjs/runtime/runtime'

type AdbInfo = {
  available: boolean
  path: string
  source: string
  version: string
  message: string
}

type Device = {
  serial: string
  state: string
  model: string
  product: string
  device: string
  transportId: string
}

type InstallLog = {
  level: string
  message: string
  time: string
}

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
const logPanel = ref<HTMLElement | null>(null)

const onlineDevices = computed(() => devices.value.filter(device => device.state === 'device'))
const selectedDevice = computed(() => devices.value.find(device => device.serial === selectedSerial.value))
const canInstall = computed(() => {
  return adbInfo.value.available && selectedDevice.value?.state === 'device' && apkPath.value !== '' && !installing.value
})

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
      statusText.value = 'APK 已选择'
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
    statusText.value = result.success ? '安装成功' : '安装失败'
    if (!result.success && result.error) {
      addLog('error', result.error)
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    statusText.value = '安装失败'
    addLog('error', message)
  } finally {
    installing.value = false
  }
}

onMounted(async () => {
  EventsOn('adb-install-log', (entry: InstallLog) => {
    logs.value.push(entry)
    void nextTick(() => {
      if (logPanel.value) {
        logPanel.value.scrollTop = logPanel.value.scrollHeight
      }
    })
  })

  await refreshAdb()
  if (adbInfo.value.available) {
    await refreshDevices()
  }
})

onBeforeUnmount(() => {
  EventsOff('adb-install-log')
})
</script>

<template>
  <main class="shell">
    <section class="toolbar">
      <div>
        <p class="eyebrow">APK Installer</p>
        <h1>ADB 安装台</h1>
      </div>
      <div class="status" :class="{ready: adbInfo.available}">
        <span class="status-dot"></span>
        <span>{{ statusText }}</span>
      </div>
    </section>

    <section class="workspace">
      <div class="panel control-panel">
        <div class="section-head">
          <div>
            <h2>ADB</h2>
            <p>{{ adbInfo.available ? adbInfo.version : adbInfo.message }}</p>
          </div>
          <button class="ghost-button" type="button" @click="refreshAdb">检查</button>
        </div>
        <div class="path-line" :title="adbInfo.path || adbInfo.message">
          {{ adbInfo.path || '未找到 adb' }}
        </div>
        <div class="source-badge">{{ adbInfo.source === 'bundled' ? '内置 adb' : '系统 adb' }}</div>

        <div class="section-head device-head">
          <div>
            <h2>设备</h2>
            <p>{{ onlineDevices.length }} 台可安装</p>
          </div>
          <button class="ghost-button" type="button" :disabled="loadingDevices || !adbInfo.available" @click="refreshDevices">
            {{ loadingDevices ? '刷新中' : '刷新' }}
          </button>
        </div>

        <div class="device-list">
          <button
            v-for="device in devices"
            :key="device.serial"
            class="device-row"
            :class="{active: selectedSerial === device.serial, blocked: device.state !== 'device'}"
            type="button"
            @click="selectedSerial = device.serial"
          >
            <span class="device-main">
              <strong>{{ deviceName(device) }}</strong>
              <small>{{ device.serial }}</small>
            </span>
            <span class="device-state">{{ device.state }}</span>
          </button>
          <div v-if="devices.length === 0" class="empty-state">暂无设备</div>
        </div>

        <label class="field">
          <span>APK 文件</span>
          <div class="file-input">
            <input v-model="apkPath" type="text" placeholder="选择或粘贴 .apk 路径" />
            <button type="button" @click="chooseApk">浏览</button>
          </div>
        </label>

        <button class="install-button" type="button" :disabled="!canInstall" @click="install">
          {{ installing ? '安装中...' : '安装 APK' }}
        </button>
      </div>

      <div class="panel log-panel-wrap">
        <div class="section-head">
          <div>
            <h2>日志</h2>
            <p>{{ selectedDevice ? deviceName(selectedDevice) : '等待设备' }}</p>
          </div>
          <button class="ghost-button" type="button" :disabled="installing || logs.length === 0" @click="logs = []">清空</button>
        </div>
        <div ref="logPanel" class="log-panel">
          <div v-if="logs.length === 0" class="empty-log">安装输出会显示在这里</div>
          <div v-for="(log, index) in logs" :key="index" class="log-line" :class="log.level">
            <span>{{ log.time }}</span>
            <code>{{ log.message }}</code>
          </div>
        </div>
      </div>
    </section>
  </main>
</template>
