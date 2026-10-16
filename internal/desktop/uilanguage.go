package desktop

// trayLanguageLabels are native menu strings, separate from daemon/session
// state. The GUI tells the host its effective UI locale after WebView loads.
type trayLanguageLabels struct {
	tooltip        string
	show           string
	quit           string
	stop           string
	stopAll        string
	stopAllTitle   string
	stopAllMessage string
	stopAllConfirm string
	stopAllCancel  string
	stopAllBusy    string
}

func trayLabelsFor(locale string) trayLanguageLabels {
	if locale == "zh" {
		return trayLanguageLabels{
			tooltip:        "DiggerTerm · 终端工作区",
			show:           "显示 DiggerTerm",
			quit:           "仅退出 DiggerTerm",
			stop:           "停止空闲后台服务并退出",
			stopAll:        "停止所有会话并退出",
			stopAllTitle:   "确认停止所有会话",
			stopAllMessage: "这将终止 %d 个由 DiggerTerm 托管的运行会话，然后停止后台服务并退出。此操作无法撤销。",
			stopAllConfirm: "终止会话并退出",
			stopAllCancel:  "取消",
			stopAllBusy:    "正在停止所有会话。",
		}
	}
	return trayLanguageLabels{
		tooltip:        "DiggerTerm · terminal workspace",
		show:           "Show DiggerTerm",
		quit:           "Quit DiggerTerm",
		stop:           "Stop idle background service and quit",
		stopAll:        "Stop all sessions and quit",
		stopAllTitle:   "Confirm stopping all sessions",
		stopAllMessage: "This will terminate %d running sessions managed by DiggerTerm, then stop the background service and quit. This cannot be undone.",
		stopAllConfirm: "Terminate sessions and quit",
		stopAllCancel:  "Cancel",
		stopAllBusy:    "A stop-all operation is already in progress.",
	}
}
