package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const monitorScript = `#!/bin/bash
APP_NAME="{{.AppPath}}"

# 监控循环
while true; do
    # 检查程序是否在运行
    if ! pgrep -f "$APP_NAME" > /dev/null; then
        echo "$(date): 程序未在运行，正在重启..."
        # 重新启动程序
        nohup $APP_NAME &
    fi
    # 每隔30秒检查一次
    sleep 30
done
`

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.MonitorScriptPath}}</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`

// 检查Wi-Fi是否开启
func isWiFiEnabled() (bool, error) {
	cmd := exec.Command("networksetup", "-getairportpower", "en0")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), "On"), nil
}

// 开启Wi-Fi
func enableWiFi() error {
	cmd := exec.Command("networksetup", "-setairportpower", "en0", "on")
	return cmd.Run()
}

// 自动化主函数
func main() {
	// 获取程序的当前路径
	appPath, err := os.Executable()
	if err != nil {
		fmt.Println("无法获取程序路径:", err)
		return
	}

	// 设置文件路径
	homeDir, _ := os.UserHomeDir()
	monitorScriptPath := filepath.Join(homeDir, "monitor.sh")
	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.example.wifiprotector.plist")

	// 创建监控脚本
	err = createMonitorScript(monitorScriptPath, appPath)
	if err != nil {
		fmt.Println("创建监控脚本失败:", err)
		return
	}

	// 创建启动项plist文件
	err = createPlistFile(plistPath, monitorScriptPath)
	if err != nil {
		fmt.Println("创建启动项文件失败:", err)
		return
	}

	// 设置监控脚本为可执行
	err = os.Chmod(monitorScriptPath, 0755)
	if err != nil {
		fmt.Println("无法设置监控脚本的权限:", err)
		return
	}

	// 加载launchd启动项
	err = loadLaunchAgent(plistPath)
	if err != nil {
		fmt.Println("加载启动项失败:", err)
		return
	}

	fmt.Println("自我保护程序已安装并设置为开机启动。")

	// 启动Wi-Fi检查循环
	for {
		enabled, err := isWiFiEnabled()
		if err != nil {
			fmt.Printf("检查Wi-Fi状态时出错: %v\n", err)
		} else if !enabled {
			fmt.Println("Wi-Fi未开启，正在尝试开启...")
			if err := enableWiFi(); err != nil {
				fmt.Printf("打开Wi-Fi时出错: %v\n", err)
			} else {
				fmt.Println("Wi-Fi已成功开启。")
			}
		} else {
			fmt.Println("Wi-Fi已开启。")
		}
		time.Sleep(10 * time.Second)
	}
}

// 创建监控脚本
func createMonitorScript(scriptPath, appPath string) error {
	// 创建脚本内容
	tmpl, err := template.New("monitor").Parse(monitorScript)
	if err != nil {
		return err
	}

	// 创建或覆盖文件
	file, err := os.Create(scriptPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 渲染并写入文件
	return tmpl.Execute(file, struct{ AppPath string }{AppPath: appPath})
}

// 创建启动项plist文件
func createPlistFile(plistPath, monitorScriptPath string) error {
	// 创建plist内容
	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return err
	}

	// 创建或覆盖文件
	file, err := os.Create(plistPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 渲染并写入文件
	return tmpl.Execute(file, struct {
		Label             string
		MonitorScriptPath string
	}{
		Label:             "com.example.wifiprotector",
		MonitorScriptPath: monitorScriptPath,
	})
}

// 加载launchd启动项
func loadLaunchAgent(plistPath string) error {
	cmd := exec.Command("launchctl", "load", plistPath)
	return cmd.Run()
}
