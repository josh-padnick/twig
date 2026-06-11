package setup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/josh-padnick/twig/internal/testutil"
)

func TestFindManifest(t *testing.T) {
	f := testutil.StandardFixture(t)

	t.Run("no manifest anywhere", func(t *testing.T) {
		_, err := FindManifest(f.Matsumoto)
		if !errors.Is(err, ErrNoManifest) {
			t.Fatalf("err = %v, want ErrNoManifest", err)
		}
	})

	// Main repo root manifest governs every worktree (committed once).
	mainManifest := filepath.Join(f.App, "twig.toml")
	if err := os.WriteFile(mainManifest, []byte("[setup]\nrun = \"true\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("worktree falls back to main repo root", func(t *testing.T) {
		got, err := FindManifest(f.Matsumoto)
		if err != nil || got != mainManifest {
			t.Errorf("got %s, err=%v, want %s", got, err, mainManifest)
		}
	})

	t.Run("subdirectory of a worktree resolves the same", func(t *testing.T) {
		sub := filepath.Join(f.Matsumoto, "deep", "dir")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		got, err := FindManifest(sub)
		if err != nil || got != mainManifest {
			t.Errorf("got %s, err=%v, want %s", got, err, mainManifest)
		}
	})

	// A worktree's own manifest wins over the main root's.
	wtManifest := filepath.Join(f.Matsumoto, "twig.toml")
	if err := os.WriteFile(wtManifest, []byte("[setup]\nrun = \"false\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("worktree-local manifest wins", func(t *testing.T) {
		got, err := FindManifest(f.Matsumoto)
		if err != nil || got != wtManifest {
			t.Errorf("got %s, err=%v, want %s", got, err, wtManifest)
		}
	})

	t.Run("non-repo dir checks only itself", func(t *testing.T) {
		plain := filepath.Join(f.Tmp, "plain")
		if err := os.MkdirAll(plain, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := FindManifest(plain); !errors.Is(err, ErrNoManifest) {
			t.Fatalf("err = %v, want ErrNoManifest", err)
		}
		p := filepath.Join(plain, "twig.toml")
		if err := os.WriteFile(p, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := FindManifest(plain)
		if err != nil || got != p {
			t.Errorf("got %s, err=%v, want %s", got, err, p)
		}
	})
}
