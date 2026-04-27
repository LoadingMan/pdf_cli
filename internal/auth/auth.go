package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"sort"
	"sync"

	"pdf-cli/internal/config"
	clierr "pdf-cli/internal/errors"

	"github.com/google/uuid"
)

var (
	deviceIDOnce  sync.Once
	cachedDeviceID string
)

// machineFingerprint 基于 MAC 地址 + hostname 生成机器唯一标识
func machineFingerprint() string {
	var macs []string
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(ifc.HardwareAddr) == 0 {
			continue
		}
		macs = append(macs, ifc.HardwareAddr.String())
	}
	sort.Strings(macs)
	host, _ := os.Hostname()
	seed := host + "|" + joinStrings(macs)
	if seed == "|" {
		return ""
	}
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func joinStrings(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}

func SaveToken(token string) error {
	cfg := config.Load()
	cfg.Token = token
	if cfg.DeviceID == "" {
		cfg.DeviceID = LoadDeviceID()
	}
	return cfg.Save()
}

func LoadToken() (string, error) {
	cfg := config.Load()
	if cfg.Token == "" {
		return "", clierr.AuthError("未登录", "请先执行 pdf-cli auth login")
	}
	return cfg.Token, nil
}

func LoadDeviceID() string {
	deviceIDOnce.Do(func() {
		cfg := config.Load()
		if cfg.DeviceID != "" {
			cachedDeviceID = cfg.DeviceID
			return
		}
		id := machineFingerprint()
		if id == "" {
			id = uuid.New().String()
		}
		cfg.DeviceID = id
		_ = cfg.Save()
		cachedDeviceID = id
	})
	return cachedDeviceID
}

// EnsureDeviceID 确保 deviceId 存在（登录前调用）
func EnsureDeviceID() string {
	return LoadDeviceID()
}

func ClearToken() error {
	cfg := config.Load()
	cfg.Token = ""
	return cfg.Save()
}

func IsLoggedIn() bool {
	cfg := config.Load()
	return cfg.Token != ""
}
