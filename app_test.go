package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractApksFromBundle(t *testing.T) {
	tempDir := t.TempDir()
	xapkPath := filepath.Join(tempDir, "sample.xapk")
	writeTestZip(t, xapkPath, map[string]string{
		"base.apk":             "base",
		"split_config.arm.apk": "arm",
		"Android/obb/data.obb": "obb",
		"manifest.json":        "{}",
		"nested/feature.apk":   "feature",
		"nested/readme.txt":    "ignored",
	})

	extractDir := filepath.Join(tempDir, "extract")
	apkFiles, err := extractApksFromBundle(xapkPath, extractDir, "XAPK")
	if err != nil {
		t.Fatalf("extractApksFromBundle() error = %v", err)
	}
	if len(apkFiles) != 3 {
		t.Fatalf("extractApksFromBundle() found %d APKs, want 3", len(apkFiles))
	}

	for _, apkPath := range apkFiles {
		if filepath.Ext(apkPath) != ".apk" {
			t.Fatalf("extracted non-APK file: %s", apkPath)
		}
		if _, err := os.Stat(apkPath); err != nil {
			t.Fatalf("extracted APK is not accessible: %v", err)
		}
	}
}

func TestIsSupportedPackageExt(t *testing.T) {
	supported := []string{".apk", ".xapk", ".apkm", ".apks"}
	for _, ext := range supported {
		if !isSupportedPackageExt(ext) {
			t.Fatalf("isSupportedPackageExt(%q) = false, want true", ext)
		}
	}

	if isSupportedPackageExt(".zip") {
		t.Fatal("isSupportedPackageExt(\".zip\") = true, want false")
	}
}

func TestSafeZipTargetPathRejectsTraversal(t *testing.T) {
	_, err := safeZipTargetPath(t.TempDir(), "../evil.apk")
	if err == nil {
		t.Fatal("safeZipTargetPath() error = nil, want traversal rejection")
	}
}

func TestSortApkFilesPutsBaseFirst(t *testing.T) {
	apkFiles := []string{
		filepath.Join("tmp", "split_config.arm.apk"),
		filepath.Join("tmp", "base.apk"),
		filepath.Join("tmp", "feature.apk"),
	}

	sortApkFiles(apkFiles)

	if filepath.Base(apkFiles[0]) != "base.apk" {
		t.Fatalf("first APK = %s, want base.apk", filepath.Base(apkFiles[0]))
	}
}

func writeTestZip(t *testing.T, path string, files map[string]string) {
	t.Helper()

	target, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer target.Close()

	writer := zip.NewWriter(target)
	defer writer.Close()

	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
}
