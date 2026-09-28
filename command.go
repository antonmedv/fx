package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// command is a vim-like command line command, run as :name[!] [args].
type command struct {
	// name is the vim-style name, the optional part in brackets: "w[rite]".
	// Any prefix of the full name that includes the part before the
	// brackets runs the command, so :w, :wr and :write all write.
	name string
	bang bool   // whether the command accepts ! after its name
	args string // argument placeholder for the help, e.g. "[file]"
	desc string // one-line description for the help
	// match overrides the name-based matching, e.g. for :<n>.
	match func(name string) bool
	run   func(m *model, c call) tea.Cmd
}

// call is a parsed command line: :name[!] [arg].
type call struct {
	name string // the name as typed
	bang bool
	arg  string // the rest of the line, trimmed
}

// commands lists every command line command, in the order of the help.
var commands = []command{
	{
		name:  "<n>",
		desc:  "goto line n",
		match: isLineNumber,
		run: func(m *model, c call) tea.Cmd {
			line, _ := strconv.Atoi(c.name)
			gotoLine(m, line)
			return nil
		},
	},
	{
		name: "w[rite]",
		bang: true,
		args: "[file]",
		desc: "write JSON to file, ! skips confirmation",
		run:  (*model).write,
	},
	{
		name: "q[uit]",
		desc: "quit",
		run: func(m *model, c call) tea.Cmd {
			return tea.Quit
		},
	},
}

// usage is the command as shown in the help: "w[rite][!] [file]".
func (c command) usage() string {
	s := c.name
	if c.bang {
		s += "[!]"
	}
	if c.args != "" {
		s += " " + c.args
	}
	return s
}

// matches reports whether name, as typed, runs the command.
func (c command) matches(name string) bool {
	if c.match != nil {
		return c.match(name)
	}
	short, full := splitName(c.name)
	return len(name) >= len(short) && strings.HasPrefix(full, name)
}

// splitName splits "w[rite]" into the shortest ("w") and the full ("write")
// form of the name.
func splitName(name string) (short, full string) {
	open := strings.IndexByte(name, '[')
	if open < 0 {
		return name, name
	}
	short = name[:open]
	full = short + strings.TrimSuffix(name[open+1:], "]")
	return short, full
}

func isLineNumber(name string) bool {
	_, err := strconv.Atoi(name)
	return err == nil
}

// parseCall splits a command line into its name, bang and argument. The name
// is the leading run of letters, or a number for :<n>.
func parseCall(line string) call {
	line = strings.TrimSpace(line)
	end := 0
	for end < len(line) && isNameByte(line[end], end) {
		end++
	}
	c := call{name: line[:end]}
	rest := line[end:]
	if strings.HasPrefix(rest, "!") {
		c.bang = true
		rest = rest[1:]
	}
	c.arg = strings.TrimSpace(rest)
	return c
}

func isNameByte(b byte, pos int) bool {
	switch {
	case 'a' <= b && b <= 'z', 'A' <= b && b <= 'Z':
		return true
	case '0' <= b && b <= '9':
		return true
	case b == '-':
		return pos == 0 // a negative line number
	}
	return false
}

// findCommand returns the command run by name, or nil if there is none.
func findCommand(name string) *command {
	for i := range commands {
		if commands[i].matches(name) {
			return &commands[i]
		}
	}
	return nil
}

// runCommand runs a command line entered after :.
func (m *model) runCommand(line string) (tea.Model, tea.Cmd) {
	c := parseCall(line)
	if c.name == "" {
		return m, nil
	}
	cmd := findCommand(c.name)
	if cmd == nil {
		return m, m.errorf("Not an editor command: %s", line)
	}
	if c.bang && !cmd.bang {
		return m, m.errorf("No ! allowed: %s", line)
	}
	return m, cmd.run(m, c)
}

// confirmation is a yes/no question shown in the command line. y runs
// yes, any other key cancels.
type confirmation struct {
	prompt string
	yes    func() tea.Cmd
}

// ask shows a yes/no question in place of the command line.
func (m *model) ask(prompt string, yes func() tea.Cmd) tea.Cmd {
	m.confirm = &confirmation{prompt: prompt + " (y/n)", yes: yes}
	return nil
}

func (m *model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	c := m.confirm
	m.confirm = nil
	if msg.Type == tea.KeyRunes && (msg.String() == "y" || msg.String() == "Y") {
		return m, c.yes()
	}
	return m, nil
}

// message shows text in the command line until the next key press.
type message struct {
	text  string
	isErr bool
}

func (m *model) errorf(format string, args ...any) tea.Cmd {
	m.message = &message{text: fmt.Sprintf(format, args...), isErr: true}
	return nil
}

func (m *model) infof(format string, args ...any) tea.Cmd {
	m.message = &message{text: fmt.Sprintf(format, args...)}
	return nil
}
