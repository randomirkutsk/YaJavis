package main

import (
	_ "embed"
	"log"
	"os"
	"os/exec"
)

//go:embed ca-certificates.crt
var embeddedCACerts []byte

func init() {
	setupCAs()
}

func setupCAs() {
	if os.Getenv("SSL_CERT_FILE") == "" {
		candidates := []string{
			"/etc/ssl/certs/ca-certificates.crt",
			"/system/vendor/quasar/ca-certificates.crt",
			"/data/quasar/ca-certificates.crt",
			"/etc/pki/tls/certs/ca-bundle.crt",
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				_ = os.Setenv("SSL_CERT_FILE", p)
				log.Printf("сеть: корневые сертификаты из %s", p)
				return
			}
		}

		// Fallback to embedded Mozilla certs
		fallbackPath := "/tmp/jarvisd/ca-certificates.crt"
		_ = os.MkdirAll("/tmp/jarvisd", 0o755)
		if len(embeddedCACerts) > 0 {
			if err := os.WriteFile(fallbackPath, embeddedCACerts, 0o644); err == nil {
				_ = os.Setenv("SSL_CERT_FILE", fallbackPath)
				log.Printf("сеть: корневые сертификаты распакованы во %s", fallbackPath)
			}
		}
	}
}

// ensureStationIntegrations — запускается в фоне при старте на колонке
func ensureStationIntegrations(dataDir string) {
	// 1. Постоянный ADB для Станции Миди (Cucumber) и других моделей
	adbMarkers := []string{
		"/data/quasar_activate_network_adb",
		"/data/quasar_activate_adb_quasmodrom",
		"/data/quasar/activate_network_adb",
		"/data/quasar/activate_adb_quasmodrom",
	}
	for _, m := range adbMarkers {
		if _, err := os.Stat(m); os.IsNotExist(err) {
			if f, err := os.Create(m); err == nil {
				f.Close()
				_ = exec.Command("chattr", "+i", m).Run()
			}
		}
	}

	// 2. Если /system/vendor/quasar/ca-certificates.crt есть, а /etc/ssl/certs нет — связываем
	if _, err := os.Stat("/system/vendor/quasar/ca-certificates.crt"); err == nil {
		if _, err := os.Stat("/etc/ssl/certs/ca-certificates.crt"); os.IsNotExist(err) {
			_ = exec.Command("mount", "-o", "remount,rw", "/").Run()
			_ = os.MkdirAll("/etc/ssl/certs", 0o755)
			_ = os.Symlink("/system/vendor/quasar/ca-certificates.crt", "/etc/ssl/certs/ca-certificates.crt")
			_ = exec.Command("mount", "-o", "remount,ro", "/").Run()
		}
	}

	// 3. Сохраняем копию сертификатов в dataDir
	if len(embeddedCACerts) > 0 && dataDir != "" {
		copyPath := dataDir + "/ca-certificates.crt"
		if _, err := os.Stat(copyPath); os.IsNotExist(err) {
			_ = os.WriteFile(copyPath, embeddedCACerts, 0o644)
		}
	}
}
