package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type worktree struct {
	name   string
	head   string
	branch string
}

func issueCommand(command string, args []string) ([]string, error) {
	cmd := exec.Command(command, args...)

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	return lines, nil
}

func parseLine(line string) worktree {
	chunks := strings.Fields(line)
	path := chunks[0]
	path_parts := strings.Split(path, "/")

	return worktree{
		name:   path_parts[len(path_parts)-1],
		head:   chunks[1],
		branch: chunks[2],
	}
}

type model struct {
	worktrees []worktree
	cursor    int
	selected  map[int]struct{}
}

func initialModel(worktrees []worktree) model {
	return model{
		worktrees: worktrees,
		selected:  make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.worktrees)-1 {
				m.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	// The header
	s := "Your worktrees\n\n"

	for i, worktree := range m.worktrees {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %-40s\t%-40s\n", cursor, checked, worktree.name, worktree.branch)
	}

	// The footer
	s += "\nPress q to quit.\n"

	return s
}

func usage() {
	fmt.Println("Usage: tree-of-work <path-to-bare-repo>")
}

func main() {

	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	}

	bareRepoPath := os.Args[1]

	git, err := exec.LookPath("git")
	if err != nil {
		log.Fatal(err)
	}

	worktreeList := []string{"-C", bareRepoPath, "worktree", "list"}
	output, _ := issueCommand(git, worktreeList)

	worktrees := make([]worktree, len(output)-2)

	for i, line := range output {
		if i == 0 || len(line) == 0 {
			continue
		}
		worktrees[i-1] = parseLine(line)
	}

	p := tea.NewProgram(initialModel(worktrees))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Coudn't run the program. Error: %v", err)
		os.Exit(1)
	}
}
