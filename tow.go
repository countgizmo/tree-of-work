package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"
)

type worktree struct {
	name       string
	head       string
	branch     string
	modifiedAt string
}

type ByModifiedAt map[int]worktree

func (a ByModifiedAt) Len() int           { return len(a) }
func (a ByModifiedAt) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByModifiedAt) Less(i, j int) bool { return a[i].modifiedAt < a[j].modifiedAt }

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

	dateArgs := []string{"-I", "-r", path}
	date, dateErr := issueCommand("date", dateArgs)
	if dateErr != nil {
		log.Fatal("date failed", dateErr)
	}

	return worktree{
		name:       path_parts[len(path_parts)-1],
		head:       chunks[1],
		branch:     chunks[2][1 : len(chunks[2])-1],
		modifiedAt: date[0],
	}
}

type model struct {
	gitPath      string
	bareRepoPath string
	worktrees    map[int]worktree
	cursor       int
	selected     map[int]struct{}
	err          error
}

func initialModel(bareRepoPath string) model {
	git, err := exec.LookPath("git")
	if err != nil {
		log.Fatal(err)
	}

	return model{
		cursor:       0,
		gitPath:      git,
		bareRepoPath: bareRepoPath,
		selected:     make(map[int]struct{}),
	}
}

type deleteMsg int
type errMsg struct{ err error }
type listMsg map[int]worktree

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

			removeBranch := []string{"-C", m.bareRepoPath, "branch", "-d", tree.branch}
			_, removeBranchErr := issueCommand(m.gitPath, removeBranch)
			if removeBranchErr != nil {
				return errMsg{removeBranchErr}
			}
		}

		return deleteMsg(0)
	}
}

func listTrees(git string, bareRepoPath string) tea.Cmd {
	return func() tea.Msg {
		worktreeList := []string{"-C", bareRepoPath, "worktree", "list"}
		output, err := issueCommand(git, worktreeList)

		if err != nil {
			return errMsg{err}
		}

		worktrees := make(map[int]worktree, len(output)-2)

		for i, line := range output {
			if i == 0 || len(line) == 0 {
				continue
			}
			worktrees[i-1] = parseLine(line)
		}

		sort.Sort(ByModifiedAt(worktrees))

		return listMsg(worktrees)
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

	// After delete operations ran, we need to update
	// the model accordingly otherwise the view will break.
	case deleteMsg:
		for k := range m.selected {
			delete(m.selected, k)
			delete(m.worktrees, k)
		}
		if m.cursor >= len(m.worktrees) {
			m.cursor = len(m.worktrees) - 1
		}

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

var sizeCmd = []string{"stty", "size"}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nWe had some trouble: %v\n\n", m.err)
	}

	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, sizeErr := cmd.Output()

	var rows = 40
	var columns = 80

	if sizeErr == nil {
		fields := strings.Fields(string(out))
		rows, _ = strconv.Atoi(fields[0])
		columns, _ = strconv.Atoi(fields[1])
		log.Printf("rows = %d columns = %d\n", rows, columns)
	}

	// The header
	current := m.cursor + 1
	if len(m.worktrees) == 0 {
		current = 0
	}

	s := fmt.Sprintf("Your worktrees: [%d/%d]\n\n", current, len(m.worktrees))

	// TODO(evgheni): refactor to render* functions for each part of the view
	//                aka: subviews?
	// The table
	var tabStrings strings.Builder
	tableWriter := tabwriter.NewWriter(&tabStrings, 20, 4, 2, '\t', 0)

	dataRows := rows - 5
	start := 0
	end := len(m.worktrees)

	if end > 0 && dataRows < len(m.worktrees) {
		end = dataRows
		if m.cursor >= dataRows {
			offset := (m.cursor + 1) - dataRows
			if offset > 0 {
				start = start + offset
				end = start + dataRows
			}
		}
	}

	for i := start; i < end; i++ {
		worktree := m.worktrees[i]

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
		fmt.Fprintf(tableWriter, "%s [%s] %s\t%s\t%s\t\n", cursor, checked, worktree.name, worktree.branch, worktree.modifiedAt)
	}
	tableWriter.Flush()
	s += tabStrings.String()

	// The footer
	s += "\nq: Quit, Enter/Space: Select, d: Delete, r: Refresh\n"

	return s
}

// TODO(evgheni): if no path is specified try the current directory.
//
//	If it's not a bare directory _than_ print out the usage.
//	Also update the usage message then.
//
// This can be useful if the tow is in path and you are in your bare repo already
// instead of calling `git worktree` you call `tow` and that's it.
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
