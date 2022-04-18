package reducer

import (
	_ "embed"
	"fmt"
	"os/exec"
)

func CreatePython(bin string, args []string) *exec.Cmd {
	cmd := exec.Command(bin, "-c", GenerateCode(args))
	return cmd
}

//go:embed reduce.py
var templatePython string

func GenerateCodePython(args []string) string {
	rs := "\n"
	for i, a := range args {
		rs += "try:"
		switch {
		case a == ".":
			rs += `
    x = x
`

		default:
			rs += fmt.Sprintf(
				`
    f = (lambda json: (%v))(x)
    x = f(x) if callable(f) else f
`, a)
		}
		// Generate a beautiful error message.
		rs += "except Exception as e:\n"
		pre, post, pointer := trace(args, i)
		rs += fmt.Sprintf(
			`    sys.stderr.write('\n  {} {} {}\n  %v\n\n{}\n'.format(%q, %q, %q, e))
    sys.exit(1)`,
			pointer,
			pre, a, post,
		)
		rs += "\n"
	}
	return fmt.Sprintf(templatePython, rs)
}
