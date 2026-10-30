package embedding

import (
	"runtime"
	"testing"
)

func TestRuntimePresetIDMatchesPlatform(t *testing.T) {
	preset, ok := RuntimePresetID()
	if !ok {
		t.Fatalf("expected a runtime preset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	for _, rt := range BuiltinCatalog().Runtimes {
		if rt.PresetID == preset {
			if rt.GOOS != runtime.GOOS || rt.GOArch != runtime.GOARCH {
				t.Fatalf("preset %s does not match current platform", preset)
			}
			return
		}
	}
	t.Fatalf("preset %q missing from catalog", preset)
}
