package engine

import (
	"math/big"
	"reflect"
	"regexp"
	"time"
)

var (
	bigIntType   = reflect.TypeOf((*big.Int)(nil))
	timeTimeType = reflect.TypeOf((*time.Time)(nil)).Elem()
)

var syntaxErrorRe = regexp.MustCompile(`^SyntaxError: SyntaxError: \(anonymous\): Line \d+:\d+\s+`)

func extractErrorMessage(input string) string {
	return syntaxErrorRe.ReplaceAllString(input, "")
}
