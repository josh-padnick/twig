package cli

import (
	"reflect"
	"testing"

	"github.com/josh-padnick/twig/internal/config"
	"github.com/josh-padnick/twig/internal/setup"
)

func TestOpenerSetPrecedence(t *testing.T) {
	global := config.Open{
		Default: []string{"ghostty"},
		Openers: map[string]config.Opener{
			"cursor": {Kind: "command", Command: "cursor {{dir}}"},
		},
	}
	repo := &setup.Loaded{Manifest: setup.Manifest{Open: config.Open{
		Default: []string{"ghostty", "cursor", "dev-browser"},
		Openers: map[string]config.Opener{
			"dev-browser": {Kind: "command", Command: "open http://localhost:5173"},
		},
	}}}

	t.Run("global default when no manifest", func(t *testing.T) {
		names, _ := openerSet(nil, global, nil, false)
		if !reflect.DeepEqual(names, []string{"ghostty"}) {
			t.Errorf("names = %v", names)
		}
	})

	t.Run("trusted repo overrides default and overlays catalog", func(t *testing.T) {
		names, catalog := openerSet(nil, global, repo, true)
		if !reflect.DeepEqual(names, []string{"ghostty", "cursor", "dev-browser"}) {
			t.Errorf("names = %v", names)
		}
		if _, ok := catalog.Openers["dev-browser"]; !ok {
			t.Error("repo-defined opener missing from catalog")
		}
		if _, ok := catalog.Openers["cursor"]; !ok {
			t.Error("global opener lost in overlay")
		}
	})

	t.Run("untrusted repo contributes nothing", func(t *testing.T) {
		names, catalog := openerSet(nil, global, repo, false)
		if !reflect.DeepEqual(names, []string{"ghostty"}) {
			t.Errorf("names = %v", names)
		}
		if _, ok := catalog.Openers["dev-browser"]; ok {
			t.Error("untrusted repo opener must not enter the catalog")
		}
	})

	t.Run("--with beats everything", func(t *testing.T) {
		names, _ := openerSet([]string{"cursor"}, global, repo, true)
		if !reflect.DeepEqual(names, []string{"cursor"}) {
			t.Errorf("names = %v", names)
		}
	})
}
