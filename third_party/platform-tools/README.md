# Platform Tools

Place platform-specific Android platform-tools here before running `wails build`.

Expected layout:

```text
third_party/platform-tools/darwin/adb
third_party/platform-tools/windows/adb.exe
third_party/platform-tools/windows/AdbWinApi.dll
third_party/platform-tools/windows/AdbWinUsbApi.dll
```

The post-build hook copies the current platform directory into the packaged app:

- macOS: `apk-install.app/Contents/Resources/platform-tools/`
- Windows: `build/bin/platform-tools/`
