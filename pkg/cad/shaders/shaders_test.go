package shaders

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestShadersCompile(t *testing.T) {
	shaders := []string{
		"terrain.wgsl",
	}

	for _, shader := range shaders {
		t.Run(shader, func(t *testing.T) {
			path := filepath.Join(".", shader)
			_, err := os.Stat(path)
			if err != nil {
				t.Fatalf("Shader %s not found: %v", shader, err)
			}
            
			// If naga is installed, compile it
			nagaPath, err := exec.LookPath("naga")
			if err == nil {
				cmd := exec.Command(nagaPath, path)
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("Naga compilation failed for %s: %v\nOutput: %s", shader, err, output)
				}
			} else {
                t.Logf("naga not found, skipping strict compilation check for %s", shader)
            }
		})
	}
}
