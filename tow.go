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
		branch: chunks[2][1:len(chunks[2])-1],
	}
}

type model struct {
	gitPath  string
	bareRepoPath string
	worktrees []worktree
	cursor    int
	selected  map[int]struct{}
	err error
}

func initialModel(bareRepoPath string) model {
	git, err := exec.LookPath("git")
	if err != nil {
		log.Fatal(err)
	}

	return model{
		gitPath: git,
		bareRepoPath: bareRepoPath,
		selected:  make(map[int]struct{}),
	}
}

type okMsg int
type errMsg struct{err error}
type cursorUpdMsg int
type listMsg []worktree

func (e errMsg) Error() string {
	return e.err.Error()
}

func deleteTrees(m model) tea.Cmd {
	return func() tea.Msg {
		for k := range m.selected {
			tree := m.worktrees[k]
			removeWorktree := []string{"-C", m.bareRepoPath, "worktree", "remove", tree.name}
			_, removeErr := issueCommand(m.gitPath, removeWorktree)
			if removeErr != nil {
				return errMsg{removeErr}
			}

			delete(m.selected, k)

			removeBranch := []string{"-C", m.bareRepoPath, "branch", "-d", tree.branch}
			_, removeBranchErr := issueCommand(m.gitPath, removeBranch)
			if removeBranchErr != nil {
				return errMsg{removeBranchErr}
			}
		}

		return okMsg(0)
	}
}

func listTrees(git string, bareRepoPath string) tea.Cmd {
	return func() tea.Msg {
		worktreeList := []string{"-C", bareRepoPath, "worktree", "list"}
		output, err := issueCommand(git, worktreeList)

		if err != nil {
			return errMsg{err}
		}

		worktrees := make([]worktree, len(output)-2)

		for i, line := range output {
			if i == 0 || len(line) == 0 {
				continue
			}
			worktrees[i-1] = parseLine(line)
		}

		return listMsg(worktrees)
	}
}

func fixCursorPosition(m model) tea.Cmd {
	return func() tea.Msg {
		if m.cursor >= len(m.worktrees) {
			m.cursor = len(m.worktrees) - 1
		}

		return cursorUpdMsg(m.cursor)
	}
}

func (m model) Init() tea.Cmd {
	return listTrees(m.gitPath, m.bareRepoPath)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case errMsg:
		m.err = msg

	case listMsg:
		m.worktrees = msg
		return m, fixCursorPosition(m)

	case cursorUpdMsg:
		m.cursor = int(msg)

	case tea.KeyMsg:
		switch msg.String() {

		case "r":
			return m, listTrees(m.gitPath, m.bareRepoPath)

		case "d":
			return m, tea.Sequence(
				deleteTrees(m),
				listTrees(m.gitPath, m.bareRepoPath))

		case "ctrl+c", "q":
			return m, tea.Quit

			// TODO(evgheni): add scrolling
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

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
	if m.err != nil {
		return fmt.Sprintf("\nWe had some trouble: %v\n\n", m.err)
	}
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

	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	p := tea.NewProgram(initialModel(bareRepoPath))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Coudn't run the program. Error: %v", err)
		os.Exit(1)
	}
}
