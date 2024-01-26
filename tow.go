package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

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

	out, err := cmd.CombinedOutput()
	lines := strings.Split(string(out), "\n")

	if err != nil {
		return lines, err
	}

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
	errMsg       string
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
type errMsg struct {
	err error
	msg string
}
type listMsg map[int]worktree

func (e errMsg) Error() string {
	return e.err.Error()
}

// TODO(evgheni): implement FORCE deletea for capital D maybe
func deleteTrees(m model, force bool) tea.Cmd {
	return func() tea.Msg {
		for k := range m.selected {
			tree := m.worktrees[k]
			removeWorktree := []string{"-C", m.bareRepoPath, "worktree", "remove", tree.name}

			if force {
				removeWorktree = append(removeWorktree, "--force")
			}

			removeOut, removeErr := issueCommand(m.gitPath, removeWorktree)
			if removeErr != nil {
				return errMsg{removeErr, removeOut[0]}
			}

			removeBranch := []string{"-C", m.bareRepoPath, "branch", "-d", tree.branch}
			removeBranchOut, removeBranchErr := issueCommand(m.gitPath, removeBranch)
			if removeBranchErr != nil {
				return errMsg{removeBranchErr, removeBranchOut[0]}
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
			return errMsg{err, output[0]}
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
		m.errMsg = msg.msg

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
			m.errMsg = ""
			return m, listTrees(m.gitPath, m.bareRepoPath)

		case "d":
			m.errMsg = ""
			return m, tea.Sequence(
				deleteTrees(m, false),
				listTrees(m.gitPath, m.bareRepoPath),
			)

		case "D":
			m.errMsg = ""
			return m, tea.Sequence(
				deleteTrees(m, true),
				listTrees(m.gitPath, m.bareRepoPath),
			)

		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			m.errMsg = ""
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			m.errMsg = ""
			if m.cursor < len(m.worktrees)-1 {
				m.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			m.errMsg = ""
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

func getTerminalSize() (int, int) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, sizeErr := cmd.Output()

	rows := 40
	columns := 80

	if sizeErr == nil {
		fields := strings.Fields(string(out))
		rows, _ = strconv.Atoi(fields[0])
		columns, _ = strconv.Atoi(fields[1])
	}

	return rows, columns
}

func getHeader(m model) string {
	current := m.cursor + 1
	if len(m.worktrees) == 0 {
		current = 0
	}

	return fmt.Sprintf("\nYour worktrees: [%d/%d]\n\n", current, len(m.worktrees))
}

func getLongestLen(m model) int {
	result := 10 // length of a date string like 2000-10-10
	for _, tree := range m.worktrees {
		if len(tree.name) > result {
			result = len(tree.name)
		}

		if len(tree.branch) > result {
			result = len(tree.branch)
		}
	}

	return result
}

func getTable(m model) string {
	var tabStrings strings.Builder

	rows, _ := getTerminalSize()
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

	maxLen := getLongestLen(m)

	// Render table headers
	tabStrings.WriteString(fmt.Sprintf(
		"%-5s %-*s  %-*s  %-*s\n",
		"",
		maxLen, "Worktree",
		maxLen, "Branch",
		maxLen, "Modified at"))

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
		tabStrings.WriteString(
			fmt.Sprintf(
				"%s [%s] %-*s  %-*s  %-*s\n",
				cursor, checked,
				maxLen, worktree.name,
				maxLen, worktree.branch,
				maxLen, worktree.modifiedAt))
	}

	return tabStrings.String()
}

func getFooter() string {
	return "\nq: Quit, Enter/Space: Select, d: Delete, D: Force Delete, r: Refresh\n"
}

func getError(m model) string {
	if m.errMsg != "" {
		return fmt.Sprintf("\tERROR: %s\n\n", m.errMsg)
	}

	return "\n\n"
}

func (m model) View() string {

	output := getHeader(m)
	output += getError(m)
	output += getTable(m)
	output += getFooter()

	return output
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
