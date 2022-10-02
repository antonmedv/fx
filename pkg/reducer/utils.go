package reducer

import (
	"fmt"
	"regexp"
	"strings"

	. "github.com/antonmedv/fx/pkg/json"
	. "github.com/antonmedv/fx/pkg/theme"
)

func Echo(object interface{}, theme Theme) {
	fmt.Println(PrettyPrint(object, 1, theme))
}

func trace(args []string, i int) (pre, post, pointer string) {
	pre = strings.Join(args[:i], " ")
	if len(pre) > 20 {
		pre = "..." + pre[len(pre)-20:]
	}
	post = strings.Join(args[i+1:], " ")
	if len(post) > 20 {
		post = post[:20] + "..."
	}
	pointer = fmt.Sprintf(
		"%v %v %v",
		strings.Repeat(" ", len(pre)),
		strings.Repeat("^", len(args[i])),
		strings.Repeat(" ", len(post)),
	)
	return
}

var flatMapRegex = regexp.MustCompile("^(\\.\\w*)+\\[]")

func fold(s []string) string {
	if len(s) == 1 {
		return "x => x" + s[0]
	}
	obj := s[0]
	if obj == "." {
		obj = "x"
	} else {
		obj = "x" + obj
	}
	return fmt.Sprintf("x => Object.values(%v).flatMap(%v)", obj, fold(s[1:]))
}
