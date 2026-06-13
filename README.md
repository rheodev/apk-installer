# apk-install

一个基于 Wails + Vue + Go 的桌面小工具，用来通过 `adb` 给已连接的 Android 设备安装 APK。

## 功能

- 列出当前连接的设备
- 选择 APK 文件
- 选择目标设备并执行安装
- 显示安装过程日志和失败原因
- `adb` 优先使用内置路径，找不到时回退到系统 `PATH`

## 运行

```bash
wails dev
```

开发模式下前端窗口会以 800x600 打开。

## 构建

```bash
wails build
```

如果希望打包内置 `adb`（避免依赖系统 PATH），构建前把对应平台的 platform-tools 放到 `third_party/platform-tools/`：

```text
third_party/platform-tools/darwin/adb
third_party/platform-tools/windows/adb.exe
third_party/platform-tools/windows/AdbWinApi.dll
third_party/platform-tools/windows/AdbWinUsbApi.dll
```

构建钩子会自动把它们复制进打包产物（macOS 放入 `.app/Contents/Resources/platform-tools/`，Windows 放入安装目录的 `platform-tools/`）。

## 前置条件

- 已安装 Android Platform Tools，或随应用内置 `adb`（见上「构建」）
- 设备已开启 USB 调试并完成授权

## `adb` 约定

程序优先使用内置 `adb`，找不到时回退到系统 `PATH`。按下面顺序逐个查找，命中第一个即使用：

1. 可执行文件同目录（开发模式下为项目根目录）
2. `<目录>/bin/adb`
3. `<目录>/platform-tools/adb`
4. 仅 macOS：`../Resources/` 及其下的 `bin`、`platform-tools`（适配 `.app` 包结构）
5. 系统 `PATH` 中的 `adb`

## 技术栈

- Go 1.23
- Wails 2.12
- Vue 3 + Vite + TypeScript
