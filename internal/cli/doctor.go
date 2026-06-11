// `twig doctor` diagnoses the environment: git, config (including its
// warnings), roots, providers, openers, the trust store, and terminal
// availability. Every line is ok/warn/fail so a glance shows what needs
// fixing; doctor itself always exits 0 unless it cannot even run.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/josh-padnick/twig/internal/config"
	"github.com/josh-padnick/twig/internal/gitx"
	"github.com/josh-padnick/twig/internal/opener"
	"github.com/josh-padnick/twig/internal/resolve"
	"github.com/josh-padnick/twig/internal/trust"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose twig's configuration and environment",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runDoctor(cmd.OutOrStdout())
			return nil
		},
	}
}

// report prints one aligned diagnostic line.
func report(w io.Writer, status, format string, args ...any) {
	fmt.Fprintf(w, "%-5s %s\n", status, fmt.Sprintf(format, args...))
}

func runDoctor(w io.Writer) {
	report(w, "ok", "twig %s", buildVersion)

	if v, err := gitx.Version(); err == nil {
		report(w, "ok", "%s", v)
	} else {
		report(w, "fail", "git not usable: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		report(w, "fail", "cannot determine home directory: %v", err)
		return
	}

	cfgPath, _ := config.Path()
	cfg, warnings, err := config.Load()
	switch {
	case err != nil:
		report(w, "fail", "config %s: %v", cfgPath, err)
	default:
		if _, statErr := os.Stat(cfgPath); statErr != nil {
			report(w, "ok", "config: none at %s (using defaults)", cfgPath)
		} else {
			report(w, "ok", "config: %s", cfgPath)
		}
	}
	for _, warn := range warnings {
		report(w, "warn", "config: %s", warn)
	}

	roots := cfg.ExpandedRoots(home)
	if len(roots) == 0 {
		report(w, "ok", "roots: none configured (providers and the current repo still work)")
	}
	for _, root := range roots {
		if fi, err := os.Stat(root); err == nil && fi.IsDir() {
			report(w, "ok", "root: %s", root)
		} else {
			report(w, "warn", "root does not exist: %s", root)
		}
	}

	for _, prov := range resolve.ByNames(cfg.Providers) {
		parents := 0
		for _, p := range prov.Parents(home, roots) {
			if fi, err := os.Stat(p); err == nil && fi.IsDir() {
				parents++
			}
		}
		if parents > 0 {
			report(w, "ok", "provider %s: %d scan location(s)", prov.Name, parents)
		} else {
			report(w, "ok", "provider %s: nothing on disk (inactive)", prov.Name)
		}
	}

	for _, name := range cfg.Open.Default {
		op, err := opener.FromConfig(name, cfg.Open)
		if err != nil {
			report(w, "warn", "opener %s: %v", name, err)
			continue
		}
		if err := op.Available(); err != nil {
			report(w, "warn", "opener %s: %v", name, err)
		} else {
			report(w, "ok", "opener %s: available", name)
		}
	}

	if store, err := trust.NewStore(); err == nil {
		entries, _ := store.List()
		report(w, "ok", "trust store: %s (%d approval(s))", store.Dir, len(entries))
	} else {
		report(w, "fail", "trust store: %v", err)
	}

	if tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
		tty.Close()
		report(w, "ok", "terminal: /dev/tty available (interactive picker and prompts work)")
	} else {
		report(w, "warn", "terminal: no /dev/tty — picker and confirmations unavailable here")
	}
}
