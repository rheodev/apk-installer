package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const installLogEvent = "adb-install-log"

type App struct {
	ctx context.Context

	// currentInstallMu 保护 currentCancel。同一时刻只允许一个安装在进行，
	// CancelApk 通过它取消正在进行的安装。
	currentInstallMu sync.Mutex
	currentCancel    context.CancelFunc
}

type AdbInfo struct {
	Available bool   `json:"available"`
	Path      string `json:"path"`
	Source    string `json:"source"`
	Version   string `json:"version"`
	Message   string `json:"message"`
}

type Device struct {
	Serial      string `json:"serial"`
	State       string `json:"state"`
	Model       string `json:"model"`
	Product     string `json:"product"`
	Device      string `json:"device"`
	TransportID string `json:"transportId"`
}

type InstallRequest struct {
	DeviceSerial string `json:"deviceSerial"`
	ApkPath      string `json:"apkPath"`
}

type InstallResult struct {
	Success   bool   `json:"success"`
	Cancelled bool   `json:"cancelled"`
	Output    string `json:"output"`
	Error     string `json:"error"`
}

type InstallLog struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetAdbInfo() AdbInfo {
	path, source, err := findAdb()
	if err != nil {
		return AdbInfo{
			Available: false,
			Message:   err.Error(),
		}
	}

	out, err := runAdb(path, 5*time.Second, "version")
	if err != nil {
		return AdbInfo{
			Available: false,
			Path:      path,
			Source:    source,
			Message:   err.Error(),
		}
	}

	return AdbInfo{
		Available: true,
		Path:      path,
		Source:    source,
		Version:   firstNonEmptyLine(out),
		Message:   "adb 可用",
	}
}

func (a *App) ListDevices() ([]Device, error) {
	path, _, err := findAdb()
	if err != nil {
		return nil, err
	}

	out, err := runAdb(path, 15*time.Second, "devices", "-l")
	if err != nil {
		return nil, err
	}

	return parseDevices(out), nil
}

func (a *App) SelectApk() (string, error) {
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择 APK 文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Android Package (*.apk)", Pattern: "*.apk"},
		},
	})
}

// InstallApk 在目标设备上执行 adb install，实时把输出推送到前端。
// 安装过程可通过 CancelApk 取消。同一时刻只允许一个安装在进行，
// 重复调用会返回错误。
func (a *App) InstallApk(req InstallRequest) (InstallResult, error) {
	serial := strings.TrimSpace(req.DeviceSerial)
	apkPath := strings.TrimSpace(req.ApkPath)

	if serial == "" {
		return InstallResult{}, errors.New("请选择一个设备")
	}
	if apkPath == "" {
		return InstallResult{}, errors.New("请选择 APK 文件")
	}
	if !strings.EqualFold(filepath.Ext(apkPath), ".apk") {
		return InstallResult{}, errors.New("请选择 .apk 文件")
	}
	info, err := os.Stat(apkPath)
	if err != nil {
		return InstallResult{}, fmt.Errorf("APK 文件不可访问: %w", err)
	}
	if info.IsDir() {
		return InstallResult{}, errors.New("APK 路径不能是目录")
	}

	adbPath, _, err := findAdb()
	if err != nil {
		return InstallResult{}, err
	}

	// 注册本次安装的取消句柄。若已有安装在进行，则拒绝。
	ctx, cancel, busyErr := a.beginInstall()
	if busyErr != nil {
		return InstallResult{}, busyErr
	}
	defer a.endInstall()
	defer cancel() // 确保 context 及其子 context 释放资源

	// 在可取消 context 上派生超时，两种来源都能终止进程。
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 15*time.Minute)
	defer timeoutCancel()

	a.emitLog("info", "开始安装: "+filepath.Base(apkPath))
	cmd := exec.CommandContext(timeoutCtx, adbPath, "-s", serial, "install", "-r", apkPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return InstallResult{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		// stdout 已打开，必须关闭，避免泄漏。
		_ = stdout.Close()
		cancel()
		return InstallResult{}, err
	}
	defer func() {
		_ = stdout.Close()
		_ = stderr.Close()
	}()

	var output bytes.Buffer
	var outputMu sync.Mutex
	appendLine := func(level string, line string) {
		outputMu.Lock()
		output.WriteString(line)
		output.WriteByte('\n')
		outputMu.Unlock()
		a.emitLog(level, line)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return InstallResult{}, err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go scanLines(&wg, stdout, func(line string) { appendLine("stdout", line) })
	go scanLines(&wg, stderr, func(line string) { appendLine("stderr", line) })
	wg.Wait()

	waitErr := cmd.Wait()

	switch {
	case errors.Is(timeoutCtx.Err(), context.Canceled):
		a.emitLog("error", "安装已取消")
		return InstallResult{Success: false, Cancelled: true, Output: output.String(), Error: "安装已取消"}, nil
	case errors.Is(timeoutCtx.Err(), context.DeadlineExceeded):
		a.emitLog("error", "安装超时")
		return InstallResult{Success: false, Output: output.String(), Error: "安装超时"}, nil
	case waitErr != nil:
		msg := waitErr.Error()
		a.emitLog("error", "安装失败: "+msg)
		return InstallResult{Success: false, Output: output.String(), Error: msg}, nil
	}

	a.emitLog("success", "安装完成")
	return InstallResult{Success: true, Output: output.String()}, nil
}

// CancelApk 取消正在进行的安装。没有进行中的安装时返回 false。
func (a *App) CancelApk() bool {
	a.currentInstallMu.Lock()
	cancel := a.currentCancel
	a.currentInstallMu.Unlock()

	if cancel == nil {
		return false
	}
	cancel()
	a.emitLog("info", "正在取消安装...")
	return true
}

// beginInstall 占用安装槽位并返回可取消的 context。
// 若已有安装在进行，返回错误。
func (a *App) beginInstall() (context.Context, context.CancelFunc, error) {
	a.currentInstallMu.Lock()
	defer a.currentInstallMu.Unlock()

	if a.currentCancel != nil {
		return nil, nil, errors.New("已有安装任务在进行中")
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.currentCancel = cancel
	return ctx, cancel, nil
}

// endInstall 释放安装槽位。
func (a *App) endInstall() {
	a.currentInstallMu.Lock()
	a.currentCancel = nil
	a.currentInstallMu.Unlock()
}

func (a *App) emitLog(level string, message string) {
	if a.ctx == nil {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, installLogEvent, InstallLog{
		Level:   level,
		Message: message,
		Time:    time.Now().Format("15:04:05"),
	})
}

func findAdb() (string, string, error) {
	name := "adb"
	if runtime.GOOS == "windows" {
		name = "adb.exe"
	}

	for _, candidate := range bundledAdbCandidates(name) {
		if isExecutableFile(candidate) {
			return candidate, "bundled", nil
		}
	}

	path, err := exec.LookPath("adb")
	if err == nil {
		return path, "system", nil
	}

	return "", "", errors.New("未找到 adb，请放入内置 adb 或安装 Android Platform Tools")
}

func bundledAdbCandidates(name string) []string {
	var bases []string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		bases = append(bases, exeDir, filepath.Join(exeDir, "bin"), filepath.Join(exeDir, "platform-tools"))

		if runtime.GOOS == "darwin" {
			resources := filepath.Clean(filepath.Join(exeDir, "..", "Resources"))
			bases = append(bases, resources, filepath.Join(resources, "bin"), filepath.Join(resources, "platform-tools"))
		}
	}
	if wd, err := os.Getwd(); err == nil {
		bases = append(bases, wd, filepath.Join(wd, "bin"), filepath.Join(wd, "platform-tools"))
	}

	candidates := make([]string, 0, len(bases))
	for _, base := range bases {
		candidates = append(candidates, filepath.Join(base, name))
	}
	return candidates
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func runAdb(adbPath string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, adbPath, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), errors.New("adb 执行超时")
	}
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			text = err.Error()
		}
		return string(out), fmt.Errorf("adb 执行失败: %s", text)
	}
	return string(out), nil
}

func parseDevices(output string) []Device {
	lines := strings.Split(output, "\n")
	devices := make([]Device, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		device := Device{
			Serial: fields[0],
			State:  fields[1],
		}
		for _, field := range fields[2:] {
			key, value, ok := strings.Cut(field, ":")
			if !ok {
				continue
			}
			switch key {
			case "model":
				device.Model = value
			case "product":
				device.Product = value
			case "device":
				device.Device = value
			case "transport_id":
				device.TransportID = value
			}
		}
		devices = append(devices, device)
	}

	return devices
}

func scanLines(wg *sync.WaitGroup, reader interface{ Read([]byte) (int, error) }, onLine func(string)) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			onLine(line)
		}
	}
}

func firstNonEmptyLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
