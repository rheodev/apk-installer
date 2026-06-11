# Repository Guidelines

## Project Structure & Module Organization

This repository is a Wails desktop app for installing APK files on connected Android devices through `adb`.

- `main.go` and `app.go`: Go entry point and backend application logic exposed to Wails.
- `frontend/src/`: Vue 3 + TypeScript source, styles, images, and fonts.
- `frontend/wailsjs/`: generated Wails bindings. Do not edit by hand.
- `build/`: platform build metadata, icons, manifests, and installer files.
- `wails.json`: Wails project configuration and frontend build commands.

## Build, Test, and Development Commands

- `wails dev`: run the desktop app in development mode with the Vite frontend watcher.
- `wails build`: build the packaged desktop application.
- `cd frontend && pnpm install`: install frontend dependencies when needed.
- `cd frontend && pnpm run build`: run Vue type checking and build the frontend bundle.
- `go test ./...`: run Go package tests from the repository root.

## Coding Style & Naming Conventions

Use `gofmt` for all Go files. Keep Go function and type names descriptive, and export names only when they are part of the Wails API surface.

Frontend code should follow the existing Vue single-file component style in `frontend/src/App.vue`. Use TypeScript for new frontend logic, camelCase for variables/functions, and PascalCase for Vue components.

## Testing Guidelines

There is no dedicated test suite yet. Add Go tests as `*_test.go` files next to the code they cover, and prefer table-driven tests for command parsing or `adb` path resolution. For frontend changes, run `cd frontend && pnpm run build`.

## Commit & Pull Request Guidelines

The current history uses Conventional Commit style, for example `feat: 实现设备选择、APK 安装与日志展示`. Continue using prefixes such as `feat:`, `fix:`, `docs:`, or `chore:`.

Pull requests should include a brief description, testing performed, and screenshots or short recordings for visible UI changes. Mention any changes to `adb` lookup behavior, Wails bindings, installer assets, or platform-specific build files.

## Security & Configuration Tips

Do not commit local SDK paths, device identifiers, signing secrets, or bundled third-party binaries without confirming license and distribution requirements. The app expects `adb` from the app directory, `bin/adb`, `platform-tools/adb`, or system `PATH`; keep that lookup order documented when changing it.
