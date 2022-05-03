package reducer

import (
	_ "embed"
	"fmt"
	"strings"
)

func js(args []string) string {
	rs := "\n"
	for i, a := range args {
		rs += "  try {"
		switch {
		case a == ".":
			rs += `
    x = function () 
      { return this }
    .call(x)
`

		case flatMapRegex.MatchString(a):
			code := fold(strings.Split(a, "[]"))
			rs += fmt.Sprintf(
				`
    x = (
      %v
    )(x)
`, code)

		case strings.HasPrefix(a, ".["):
			rs += fmt.Sprintf(
				`
    x = function () 
      { return this%v } 
    .call(x)
`, a[1:])

		case strings.HasPrefix(a, "."):
			rs += fmt.Sprintf(
				`
    x = function () 
      { return this%v } 
    .call(x)
`, a)

		default:
			rs += fmt.Sprintf(
				`
    let f = function () 
      { return %v }
    .call(x)
    x = typeof f === 'function' ? f(x) : f
`, a)
		}
		// Generate a beautiful error message.
		rs += "  } catch (e) {\n"
		pre, post, pointer := trace(args, i)
		rs += fmt.Sprintf(
			"    throw `\\n  ${%q} ${%q} ${%q}\\n  %v\\n\\n${e.stack || e}`\n",
			pre, a, post, pointer,
		)
		rs += "  }\n"
	}

	fn := `function reduce(input) {
  let x = JSON.parse(input)

  // Reducers %v
  if (typeof x === 'undefined') {
    return 'null'
  } else {
    return JSON.stringify(x)
  }
}
`
	return fmt.Sprintf(fn, rs)
}
