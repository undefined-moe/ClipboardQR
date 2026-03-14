# ClipboardQR

监控系统剪贴板，自动识别截图中的二维码，通过系统通知展示解码结果。

## 功能

- **剪贴板监控** — 实时监听剪贴板图片变化，自动检测新截图
- **二维码解码** — 支持 PNG / JPEG / TIFF 格式，基于 ZXing 算法
- **系统通知** — 解码成功后弹出系统通知，支持一键复制内容或打开链接
- **图片去重** — 相同截图不会重复触发通知
- **系统托盘** — 后台运行，托盘图标提供退出入口
- **跨平台** — 支持 Windows、macOS、Linux

## 平台特性

| 平台 | 剪贴板监听 | 通知方式 | 额外功能 |
|------|-----------|---------|---------|
| Windows | 系统 API | WinRT Toast | 开机自启 |
| macOS | 系统 API (含 TIFF) | terminal-notifier / osascript | — |
| Linux | X11 / Wayland | D-Bus freedesktop notifications | — |

## 安装

从 [Releases](https://github.com/undefined-moe/ClipboardQR/releases) 下载对应平台的可执行文件，直接运行即可。

### 从源码构建

```bash
# 需要 Go 1.25+，Linux 需额外安装依赖：
# sudo apt-get install libx11-dev libgtk-3-dev libayatana-appindicator3-dev

go build -trimpath -ldflags="-s -w" -o clipboardqr ./cmd/clipboardqr
```

## 使用

```bash
# 启动（后台运行，托盘图标可见）
./clipboardqr

# 启用详细日志
./clipboardqr -v
```

启动后：
1. 系统托盘出现 ClipboardQR 图标
2. 截取包含二维码的图片（或复制到剪贴板）
3. 自动弹出通知，显示解码内容
4. 点击「复制内容」复制到剪贴板，或点击「打开链接」在浏览器中打开

## License

MIT
