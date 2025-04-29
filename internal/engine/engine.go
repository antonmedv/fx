package engine

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dop251/goja"
	"github.com/goccy/go-yaml"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
)

//go:embed stdlib.js
var Stdlib string

//go:embed prelude.js
var prelude string

func Reduce(args []string) {
	if len(args) < 1 {
		panic("args must have at least one element")
	}

	var (
		flagYaml  bool
		flagRaw   bool
		flagSlurp bool
	)

	var src io.Reader = os.Stdin
	if isFile(args[0]) {
		src = open(args[0], &flagYaml)
		args = args[1:]
	} else if isFile(args[len(args)-1]) {
		src = open(args[len(args)-1], &flagYaml)
		args = args[:len(args)-1]
	}

	var fns []string
	for _, arg := range args {
		switch arg {
		case "--yaml":
			flagYaml = true
		case "--raw", "-r":
			flagRaw = true
		case "--slurp", "-s":
			flagSlurp = true
		case "-rs", "-sr":
			flagRaw = true
			flagSlurp = true
		default:
			fns = append(fns, arg)
		}
	}

	if flagSlurp {
		println("Error: Built-in JS engine does not support \"--slurp\" flag. Install Node.js or Deno to use this flag.")
		os.Exit(1)
	}

	data, err := io.ReadAll(src)
	if err != nil {
		panic(err)
	}

	if flagRaw {
		data = []byte(strconv.Quote(string(data)))
	} else if flagYaml {
		data, err = yaml.YAMLToJSON(data)
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}
	} else {
		node, err := jsonx.Parse(data)
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}
		data = []byte(node.String())
	}

	var code strings.Builder
	code.WriteString(prelude)
	code.WriteString(Stdlib)
	code.WriteString(fmt.Sprintf("let json = JSON.parse(%q)\n", data))
	for _, fn := range fns {
		code.WriteString(Transform(fn))
	}
	code.WriteString(`json === skip ? '__skip__' : JSON.stringify(json)`)

	vm := goja.New()
	vm.Set("println", func(s string) any {
		fmt.Println(s)
		return nil
	})

	value, err := vm.RunString(code.String())
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	output, ok := value.Export().(string)
	if !ok {
		println("undefined")
		return
	}

	if output == "__skip__" {
		return
	}

	node, err := jsonx.Parse([]byte(output))
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	if len(node.Value) > 0 && node.Value[0] == '"' {
		s, _ := strconv.Unquote(string(node.Value))
		fmt.Println(s)
		return
	}
	fmt.Print(theme.PrettyPrint(node))
}
