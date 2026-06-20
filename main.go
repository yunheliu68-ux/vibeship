package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbletea"

	"github.com/francis/vibeship/internal/config"
	"github.com/francis/vibeship/internal/ingest"
	"github.com/francis/vibeship/internal/model"
	"github.com/francis/vibeship/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "collect" {
		runCollect()
		return
	}
	runTUI()
}

func runCollect() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot find home dir:", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(home, ".vibeship", "data.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot create data dir:", err)
		os.Exit(1)
	}

	st, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot open store:", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := ingest.IngestStatusline(os.Stdin, st); err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: ingest error:", err)
		os.Exit(1)
	}
}

func runTUI() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vibeship: cannot find home dir:", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(home, ".vibeship", "data.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	st, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vibeship: failed to open store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	reg, _ := config.LoadSkillsAndPlugins(home)

	m := model.New(st, reg)
	defer m.Close()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vibeship: error: %v\n", err)
		os.Exit(1)
	}
}
