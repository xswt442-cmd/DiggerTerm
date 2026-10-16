package desktop

import "testing"

func TestTrayLabelsForLocale(t *testing.T) {
	for _, lang := range []string{"zh", "en"} {
		labels := trayLabelsFor(lang)
		if labels.tooltip == "" || labels.show == "" || labels.quit == "" || labels.stop == "" {
			t.Fatalf("missing %s tray labels: %+v", lang, labels)
		}
		if lang == "zh" && labels.show == trayLabelsFor("en").show {
			t.Fatal("Chinese tray menu has not been translated")
		}
		if lang == "zh" && (labels.quit != "仅退出 DiggerTerm" || labels.stop != "停止空闲后台服务并退出" || labels.stopAll != "停止所有会话并退出" || labels.stopAllCancel != "取消" || labels.stopAllConfirm != "终止会话并退出") {
			t.Fatalf("unexpected Chinese tray menu labels: %+v", labels)
		}
		if lang == "en" && (labels.stop != "Stop idle background service and quit" || labels.stopAll != "Stop all sessions and quit" || labels.stopAllCancel != "Cancel" || labels.stopAllConfirm != "Terminate sessions and quit") {
			t.Fatalf("unexpected English tray menu labels: %+v", labels)
		}
		if labels.stopAllTitle == "" || labels.stopAllMessage == "" || labels.stopAllBusy == "" {
			t.Fatalf("missing %s bulk-stop confirmation text: %+v", lang, labels)
		}
	}
}
func TestSetUILanguageValidation(t *testing.T) {
	s := &SessionService{}
	if err := s.SetUILanguage("other"); err == nil {
		t.Fatal("invalid UI locale accepted")
	}
	var got string
	s.setUILanguage = func(locale string) { got = locale }
	if err := s.SetUILanguage("zh"); err != nil {
		t.Fatal(err)
	}
	if got != "zh" {
		t.Fatalf("native tray callback got %q", got)
	}
	if err := s.SetUILanguage("en"); err != nil || got != "en" {
		t.Fatalf("English tray callback got %q, error %v", got, err)
	}
}

func TestStopAllConfirmationButtonsMatchPlatformDialogResponses(t *testing.T) {
	if confirm, cancel := stopAllConfirmationButtonLabels("zh", "windows"); confirm != "Yes" || cancel != "No" {
		t.Fatalf("Windows question button IDs = %q/%q, want Yes/No", confirm, cancel)
	}
	if confirm, cancel := stopAllConfirmationButtonLabels("zh", "linux"); confirm != "终止会话并退出" || cancel != "取消" {
		t.Fatalf("localized Linux question buttons = %q/%q", confirm, cancel)
	}
	if confirm, cancel := stopAllConfirmationButtonLabels("en", "darwin"); confirm != "Terminate sessions and quit" || cancel != "Cancel" {
		t.Fatalf("localized macOS question buttons = %q/%q", confirm, cancel)
	}
}
