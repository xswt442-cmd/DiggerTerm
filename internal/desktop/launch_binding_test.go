package desktop

import (
	"os"
	"regexp"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestFrontendLaunchBindingResolves(t *testing.T) {
	source, err := os.ReadFile("../../frontend/src/App.vue")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(?:Call\.ByName\('([^']*LaunchDirectory)'|\b(LaunchDirectory)\()`).FindSubmatch(source)
	if match == nil {
		t.Fatal("launch binding not used")
	}
	if len(match[2]) > 0 {
		return
	}
	_ = application.New(application.Options{Name: "binding test"})
	bindings := application.NewBindings(nil, nil)
	if err := bindings.Add(application.NewService(&SessionService{})); err != nil {
		t.Fatal(err)
	}
	if bindings.Get(&application.CallOptions{MethodName: string(match[1])}) == nil {
		t.Fatalf("unknown bound method name %q", match[1])
	}
}

func TestFrontendSessionCallsResolve(t *testing.T) {
	_ = application.New(application.Options{Name: "binding test"})
	bindings := application.NewBindings(nil, nil)
	if err := bindings.Add(application.NewService(&SessionService{})); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../../frontend/src/App.vue", "../../frontend/src/sshDirectoryBrowser.ts"} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		matches := regexp.MustCompile(`Call\.ByName\('([^']*SessionService\.[^']+)'`).FindAllSubmatch(source, -1)
		if len(matches) == 0 {
			t.Fatalf("no service calls in %s", path)
		}
		for _, match := range matches {
			if bindings.Get(&application.CallOptions{MethodName: string(match[1])}) == nil {
				t.Errorf("unknown bound method %q", match[1])
			}
		}
	}
}
