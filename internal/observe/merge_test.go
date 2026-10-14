package observe

import "testing"

func TestMergeObservedReservesWSL(t *testing.T) {
	win, wsl := make([]Process, 160), make([]Process, 40)
	for i := range win {
		win[i].Source = "windows"
	}
	for i := range wsl {
		wsl[i].Source = "wsl"
	}
	got := mergeObserved(win, wsl, 60)
	if len(got) != 60 {
		t.Fatalf("length=%d", len(got))
	}
	foundWSL := 0
	for _, p := range got {
		if p.Source == "wsl" {
			foundWSL++
		}
	}
	if foundWSL != 30 {
		t.Fatalf("WSL reserved=%d, wanted 30", foundWSL)
	}
	if n := len(mergeObserved(win, nil, 160)); n != 160 {
		t.Fatalf("lost slots: %d", n)
	}
	if n := len(mergeObserved(nil, wsl, 160)); n != 40 {
		t.Fatalf("lost WSL slots: %d", n)
	}
}
