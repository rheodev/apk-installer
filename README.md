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

## 前置条件

- 已安装 Android Platform Tools，或把 `adb` 放到可执行路径中
- 设备已开启 USB 调试并完成授权

## `adb` 约定

程序会按下面顺序查找 `adb`：

1. 应用程序同目录下的内置 `adb`
2. `bin/adb`
3. `platform-tools/adb`
4. 系统 `PATH` 中的 `adb`

## 技术栈

- Go 1.23
- Wails 2.12
- Vue 3 + Vite + TypeScript
