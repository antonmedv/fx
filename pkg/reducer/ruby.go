package reducer

import (
	_ "embed"
	"fmt"
	"os/exec"
)

func CreateRuby(args []string) *exec.Cmd {
	cmd := exec.Command("ruby", "-e", ruby(args))
	return cmd
}

//go:embed ruby.rb
var templateRuby string

func ruby(args []string) string {
	rs := "\n"
	for i, a := range args {
		rs += "begin"
		switch {
		case a == ".":
			rs += `
    x = x
`

		default:
			rs += fmt.Sprintf(
				`
    x = lambda {|x| %v }.call(x)
`, a)
		}
		// Generate a beautiful error message.
		rs += "rescue Exception => e\n"
		pre, post, pointer := trace(args, i)
		rs += fmt.Sprintf(
			`    STDERR.puts "\n  #{%q} #{%q} #{%q}\n  %v\n\n#{e}\n"
    exit(1)
`,
			pre, a, post,
			pointer,
		)
		rs += "end\n"
	}
	return fmt.Sprintf(templateRuby, rs)
}
