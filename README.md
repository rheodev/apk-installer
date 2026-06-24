# apk-install

一个基于 Wails + Vue + Go 的桌面小工具，用来通过 `adb` 给已连接的 Android 设备安装 APK、XAPK、APKM、APKS。

## 功能

- 列出当前连接的设备
- 选择 APK、XAPK、APKM 或 APKS 文件
- 选择目标设备并执行安装
- 自动识别 `.apk` / `.xapk` / `.apkm` / `.apks`，压缩包格式会解包后安装内部 APK
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

## CI 构建 / 发布

仓库已配置 GitHub Actions（`.github/workflows/release.yml`），可在云端构建 macOS 与 Windows 产物，无需本机交叉编译环境。

**触发方式**：推送形如 `v*` 的 tag，例如：

```bash
git tag v1.0.0
git push origin v1.0.0
```

**流程**：
1. `build-macos`：构建 `darwin/universal` 的 `.app`，打包成 zip
2. `build-windows`：构建 `windows/amd64` 的 `.exe` 及 NSIS 安装包
3. `release`：两个 job 成功后，自动创建 GitHub Release 并上传产物

两个平台各自从 Google 官方源下载对应平台的 `platform-tools` 并打包进产物，因此无需在 git 中保存 adb 二进制。`platform-tools` 不会单独发布，已随 `.app` / 安装包分发。

> 产物未经代码签名 / 公证：macOS 首次打开需在「系统设置 → 隐私与安全性」放行；Windows SmartScreen 可能提示未知发布者。

也支持在 Actions 页面手动触发（`workflow_dispatch`），但手动触发只产 artifact、不发布 Release。

> 提示：tag 名应与 `wails.json` 的 `info.productVersion` 保持一致，例如 `v1.0.0` 对应 `1.0.0`。


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

> XAPK/APKM/APKS 当前只安装包内 APK，不处理 OBB 等额外数据包。

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
