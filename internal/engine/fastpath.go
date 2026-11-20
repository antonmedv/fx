package engine

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/utils"
)

type filterOp int

const (
	opEq filterOp = iota
	opStrictEq
	opNotEq
	opStrictNotEq
	opLt
	opLte
	opGt
	opGte
	opTruthy
	opFalsy
	opIncludes
)

type valType int

const (
	valNumber valType = iota
	valString
	valBool
	valNull
	valUndefined
)

type filterCondition struct {
	path    []any
	op      filterOp
	valType valType
	numVal  float64
	strVal  string
	boolVal bool
}

type fastPathStep interface {
	apply(node *jsonx.Node) (*jsonx.Node, bool, error)
}

type identityStep struct{}

func (s *identityStep) apply(node *jsonx.Node) (*jsonx.Node, bool, error) {
	return node, true, nil
}

type lookupStep struct {
	path []any
}

func (s *lookupStep) apply(node *jsonx.Node) (*jsonx.Node, bool, error) {
	target := findByPath(node, s.path)
	if target == nil {
		return nil, false, &Error{"undefined"}
	}
	cloned := cloneSubtree(target)
	cloned.Key = ""
	cloned.Comma = false
	if cloned.End != nil {
		cloned.End.Comma = false
	}
	shiftDepth(cloned, -int(cloned.Depth))
	resetLineNumbers(cloned, 1)
	return cloned, true, nil
}

type filterStep struct {
	cond *filterCondition
}

func (s *filterStep) apply(node *jsonx.Node) (*jsonx.Node, bool, error) {
	if node.Kind == jsonx.Array {
		var matching []*jsonx.Node
		it := firstChild(node)
		for it != nil && it != node.End {
			if s.cond.Match(it) {
				matching = append(matching, it)
			}
			it = nextSiblingNode(it)
		}
		return buildArray(matching), true, nil
	}

	if s.cond.Match(node) {
		cloned := cloneSubtree(node)
		cloned.Key = ""
		cloned.Comma = false
		if cloned.End != nil {
			cloned.End.Comma = false
		}
		shiftDepth(cloned, -int(cloned.Depth))
		resetLineNumbers(cloned, 1)
		return cloned, true, nil
	}
	return nil, false, nil
}

func compileFastPath(args []string) ([]fastPathStep, bool) {
	if len(args) == 0 {
		return nil, false
	}

	steps := make([]fastPathStep, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			return nil, false
		}
		if arg == "." || arg == "x" || arg == "this" {
			steps = append(steps, &identityStep{})
			continue
		}
		if cond, ok := parseFilterCondition(arg); ok {
			steps = append(steps, &filterStep{cond: cond})
			continue
		}
		if path, ok := parseLookupPath(arg); ok {
			steps = append(steps, &lookupStep{path: path})
			continue
		}
		return nil, false
	}
	return steps, true
}

func isUndefinedErr(err error) bool {
	return err != nil && err.Error() == "undefined"
}

func runFastPath(steps []fastPathStep, parser Parser, out chan *jsonx.Node, errCh chan error, cancel <-chan struct{}, timeout time.Duration) int {
	isIdentityOnly := len(steps) == 1
	if isIdentityOnly {
		if _, ok := steps[0].(*identityStep); !ok {
			isIdentityOnly = false
		}
	}

	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timeoutChan = timer.C
	}

	for {
		select {
		case <-cancel:
			return 0
		case <-timeoutChan:
			sendErr(errCh, &Error{fmt.Sprintf("Query stopped: it took longer than %v", timeout)}, cancel)
			return 1
		default:
		}

		node, err := parser.Parse()
		if err != nil {
			if err == io.EOF {
				break
			}
			sendErr(errCh, err, cancel)
			return 1
		}

		if isIdentityOnly {
			if !send(out, node, cancel) {
				return 0
			}
			continue
		}

		if verr := node.Validate(); verr != nil {
			sendErr(errCh, verr, cancel)
			return 1
		}

		cur := node
		keep := true
		var stepErr error

		for _, step := range steps {
			cur, keep, stepErr = step.apply(cur)
			if stepErr != nil {
				sendErr(errCh, stepErr, cancel)
				if isUndefinedErr(stepErr) {
					keep = false
					break
				}
				return 1
			}
			if !keep {
				break
			}
		}

		if !keep {
			continue
		}

		if !send(out, cur, cancel) {
			return 0
		}
	}

	return 0
}

func parsePath(s string) ([]any, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	if s == "." || s == "x" || s == "this" {
		return []any{}, true
	}

	if strings.HasPrefix(s, "x.") {
		s = s[1:]
	} else if strings.HasPrefix(s, "this.") {
		s = s[4:]
	} else if strings.HasPrefix(s, "x[") {
		s = s[1:]
	} else if strings.HasPrefix(s, "this[") {
		s = s[4:]
	}

	if !strings.HasPrefix(s, ".") && !strings.HasPrefix(s, "[") {
		return nil, false
	}

	var path []any
	i := 0
	for i < len(s) {
		if s[i] == '.' {
			i++
			if i == len(s) {
				return nil, false
			}
			if s[i] == '[' {
				continue
			}
			start := i
			for i < len(s) && isIdent(s[i]) {
				i++
			}
			if i == start {
				return nil, false
			}
			path = append(path, s[start:i])
		} else if s[i] == '[' {
			i++
			if i >= len(s) {
				return nil, false
			}
			if s[i] == '"' || s[i] == '\'' {
				quote := s[i]
				i++
				start := i
				escaped := false
				for i < len(s) && (s[i] != quote || escaped) {
					if escaped {
						escaped = false
					} else if s[i] == '\\' {
						escaped = true
					}
					i++
				}
				if i >= len(s) || s[i] != quote {
					return nil, false
				}
				content := s[start:i]
				i++
				for i < len(s) && isSpace(s[i]) {
					i++
				}
				if i >= len(s) || s[i] != ']' {
					return nil, false
				}
				i++
				path = append(path, unquoteJSString(content, quote))
			} else {
				start := i
				for i < len(s) && s[i] >= '0' && s[i] <= '9' {
					i++
				}
				if i == start {
					return nil, false
				}
				idx, err := strconv.Atoi(s[start:i])
				if err != nil {
					return nil, false
				}
				for i < len(s) && isSpace(s[i]) {
					i++
				}
				if i >= len(s) || s[i] != ']' {
					return nil, false
				}
				i++
				path = append(path, idx)
			}
		} else {
			return nil, false
		}
	}

	return path, true
}

func parseLookupPath(s string) ([]any, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "." || s == "x" || s == "this" {
		return nil, false
	}
	path, ok := parsePath(s)
	if !ok || len(path) == 0 {
		return nil, false
	}
	return path, true
}

func parseFilterCondition(s string) (*filterCondition, bool) {
	s = strings.TrimSpace(s)

	if strings.Contains(s, "?") && strings.Contains(s, "skip") {
		parts := strings.Split(s, "?")
		if len(parts) == 2 {
			condPart := strings.TrimSpace(parts[0])
			rest := strings.TrimSpace(parts[1])
			colonParts := strings.Split(rest, ":")
			if len(colonParts) == 2 {
				target := strings.TrimSpace(colonParts[0])
				skipPart := strings.TrimSpace(colonParts[1])
				if (target == "x" || target == "this") && skipPart == "skip" {
					return parseFilterConditionInner(condPart)
				}
			}
		}
	}

	if strings.HasPrefix(s, "?") {
		inner := strings.TrimSpace(s[1:])
		if strings.HasPrefix(inner, "x =>") {
			inner = strings.TrimSpace(strings.TrimPrefix(inner, "x =>"))
		} else if strings.HasPrefix(inner, "(x) =>") {
			inner = strings.TrimSpace(strings.TrimPrefix(inner, "(x) =>"))
		}
		return parseFilterConditionInner(inner)
	}

	return nil, false
}

func parseFilterConditionInner(s string) (*filterCondition, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}

	if idx := strings.Index(s, ".includes("); idx >= 0 && strings.HasSuffix(s, ")") {
		left := strings.TrimSpace(s[:idx])
		arg := strings.TrimSpace(s[idx+len(".includes(") : len(s)-1])
		path, ok := parsePath(left)
		if !ok {
			return nil, false
		}
		vt, num, str, b, ok := parseLiteral(arg)
		if !ok {
			return nil, false
		}
		return &filterCondition{
			path:    path,
			op:      opIncludes,
			valType: vt,
			numVal:  num,
			strVal:  str,
			boolVal: b,
		}, true
	}

	operators := []struct {
		str string
		op  filterOp
	}{
		{"===", opStrictEq},
		{"!==", opStrictNotEq},
		{"==", opEq},
		{"!=", opNotEq},
		{"<=", opLte},
		{">=", opGte},
		{"<", opLt},
		{">", opGt},
	}

	for _, o := range operators {
		if idx := strings.Index(s, o.str); idx >= 0 {
			left := strings.TrimSpace(s[:idx])
			right := strings.TrimSpace(s[idx+len(o.str):])

			path, ok := parsePath(left)
			if !ok {
				continue
			}
			vt, num, str, b, ok := parseLiteral(right)
			if !ok {
				continue
			}
			return &filterCondition{
				path:    path,
				op:      o.op,
				valType: vt,
				numVal:  num,
				strVal:  str,
				boolVal: b,
			}, true
		}
	}

	if strings.HasPrefix(s, "!") {
		path, ok := parsePath(s[1:])
		if ok {
			return &filterCondition{
				path: path,
				op:   opFalsy,
			}, true
		}
	}

	path, ok := parsePath(s)
	if ok && len(path) > 0 {
		return &filterCondition{
			path: path,
			op:   opTruthy,
		}, true
	}

	return nil, false
}

func unquoteJSString(content string, quote byte) string {
	var b strings.Builder
	escaped := false
	for i := 0; i < len(content); i++ {
		c := content[i]
		if escaped {
			switch c {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case quote:
				b.WriteByte(quote)
			default:
				b.WriteByte(c)
			}
			escaped = false
		} else if c == '\\' {
			escaped = true
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func parseLiteral(s string) (valType, float64, string, bool, bool) {
	s = strings.TrimSpace(s)
	if s == "true" {
		return valBool, 0, "", true, true
	}
	if s == "false" {
		return valBool, 0, "", false, true
	}
	if s == "null" {
		return valNull, 0, "", false, true
	}
	if s == "undefined" {
		return valUndefined, 0, "", false, true
	}
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) || (strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		if len(s) < 2 {
			return 0, 0, "", false, false
		}
		quote := s[0]
		content := s[1 : len(s)-1]
		return valString, 0, unquoteJSString(content, quote), false, true
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return valNumber, f, "", false, true
	}

	return 0, 0, "", false, false
}

func (c *filterCondition) Match(item *jsonx.Node) bool {
	var valNode *jsonx.Node
	if len(c.path) == 0 {
		valNode = item
	} else {
		valNode = findByPath(item, c.path)
	}

	if valNode == nil {
		switch c.op {
		case opTruthy:
			return false
		case opFalsy:
			return true
		case opEq:
			return c.valType == valNull || c.valType == valUndefined
		case opStrictEq:
			return c.valType == valUndefined
		case opNotEq:
			return !(c.valType == valNull || c.valType == valUndefined)
		case opStrictNotEq:
			return c.valType != valUndefined
		case opLt, opLte, opGt, opGte:
			return false
		case opIncludes:
			return false
		}
		return false
	}

	switch c.op {
	case opTruthy:
		return !isFalsely(valNode)
	case opFalsy:
		return isFalsely(valNode)
	case opIncludes:
		return matchIncludes(valNode, c)
	case opStrictEq:
		return matchStrictEq(valNode, c)
	case opStrictNotEq:
		return !matchStrictEq(valNode, c)
	case opEq:
		return matchLooseEq(valNode, c)
	case opNotEq:
		return !matchLooseEq(valNode, c)
	case opLt:
		return matchRelational(valNode, c, func(a, b float64) bool { return a < b }, func(a, b string) bool { return a < b })
	case opLte:
		return matchRelational(valNode, c, func(a, b float64) bool { return a <= b }, func(a, b string) bool { return a <= b })
	case opGt:
		return matchRelational(valNode, c, func(a, b float64) bool { return a > b }, func(a, b string) bool { return a > b })
	case opGte:
		return matchRelational(valNode, c, func(a, b float64) bool { return a >= b }, func(a, b string) bool { return a >= b })
	}
	return false
}

func isFalsely(n *jsonx.Node) bool {
	if n == nil {
		return true
	}
	if n.Kind == jsonx.Null || n.Kind == jsonx.Undefined {
		return true
	}
	if n.Kind == jsonx.Bool && n.Value == "false" {
		return true
	}
	return false
}

func matchStrictEq(n *jsonx.Node, c *filterCondition) bool {
	switch n.Kind {
	case jsonx.Number:
		if c.valType != valNumber {
			return false
		}
		v, err := strconv.ParseFloat(n.Value, 64)
		return err == nil && v == c.numVal
	case jsonx.String:
		if c.valType != valString {
			return false
		}
		unquoted, err := utils.Unquote(n.Value)
		if err != nil {
			return false
		}
		return unquoted == c.strVal
	case jsonx.Bool:
		if c.valType != valBool {
			return false
		}
		return (n.Value == "true") == c.boolVal
	case jsonx.Null:
		return c.valType == valNull
	case jsonx.Undefined:
		return c.valType == valUndefined
	default:
		return false
	}
}

func matchLooseEq(n *jsonx.Node, c *filterCondition) bool {
	if matchStrictEq(n, c) {
		return true
	}

	if (n.Kind == jsonx.Null || n.Kind == jsonx.Undefined) && (c.valType == valNull || c.valType == valUndefined) {
		return true
	}
	if n.Kind == jsonx.Null || n.Kind == jsonx.Undefined || c.valType == valNull || c.valType == valUndefined {
		return false
	}

	if n.Kind == jsonx.Number && c.valType == valString {
		numVal, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return false
		}
		strNum, err := strconv.ParseFloat(c.strVal, 64)
		return err == nil && numVal == strNum
	}
	if n.Kind == jsonx.String && c.valType == valNumber {
		unquoted, err := utils.Unquote(n.Value)
		if err != nil {
			return false
		}
		strNum, err := strconv.ParseFloat(unquoted, 64)
		return err == nil && strNum == c.numVal
	}

	if n.Kind == jsonx.Bool {
		bNum := 0.0
		if n.Value == "true" {
			bNum = 1.0
		}
		if c.valType == valNumber {
			return bNum == c.numVal
		}
		if c.valType == valString {
			strNum, err := strconv.ParseFloat(c.strVal, 64)
			return err == nil && bNum == strNum
		}
	}
	if c.valType == valBool {
		bNum := 0.0
		if c.boolVal {
			bNum = 1.0
		}
		if n.Kind == jsonx.Number {
			nNum, err := strconv.ParseFloat(n.Value, 64)
			return err == nil && nNum == bNum
		}
		if n.Kind == jsonx.String {
			unquoted, err := utils.Unquote(n.Value)
			if err != nil {
				return false
			}
			strNum, err := strconv.ParseFloat(unquoted, 64)
			return err == nil && strNum == bNum
		}
	}

	return false
}

func matchRelational(n *jsonx.Node, c *filterCondition, cmpNum func(a, b float64) bool, cmpStr func(a, b string) bool) bool {
	if n.Kind == jsonx.String && c.valType == valString {
		unquoted, err := utils.Unquote(n.Value)
		if err != nil {
			return false
		}
		return cmpStr(unquoted, c.strVal)
	}

	numA, okA := toNumber(n)
	if !okA {
		return false
	}
	numB, okB := literalToNumber(c)
	if !okB {
		return false
	}
	return cmpNum(numA, numB)
}

func toNumber(n *jsonx.Node) (float64, bool) {
	switch n.Kind {
	case jsonx.Number:
		f, err := strconv.ParseFloat(n.Value, 64)
		return f, err == nil
	case jsonx.String:
		unquoted, err := utils.Unquote(n.Value)
		if err != nil {
			return 0, false
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(unquoted), 64)
		return f, err == nil
	case jsonx.Bool:
		if n.Value == "true" {
			return 1, true
		}
		return 0, true
	case jsonx.Null:
		return 0, true
	default:
		return 0, false
	}
}

func literalToNumber(c *filterCondition) (float64, bool) {
	switch c.valType {
	case valNumber:
		return c.numVal, true
	case valString:
		f, err := strconv.ParseFloat(strings.TrimSpace(c.strVal), 64)
		return f, err == nil
	case valBool:
		if c.boolVal {
			return 1, true
		}
		return 0, true
	case valNull:
		return 0, true
	default:
		return 0, false
	}
}

func matchIncludes(n *jsonx.Node, c *filterCondition) bool {
	if n.Kind == jsonx.String {
		if c.valType != valString {
			return false
		}
		unquoted, err := utils.Unquote(n.Value)
		if err != nil {
			return false
		}
		return strings.Contains(unquoted, c.strVal)
	}

	if n.Kind == jsonx.Array {
		it := firstChild(n)
		for it != nil && it != n.End {
			if matchStrictEq(it, c) {
				return true
			}
			it = nextSiblingNode(it)
		}
		return false
	}

	return false
}

func findByPath(n *jsonx.Node, path []any) *jsonx.Node {
	it := n
	for _, part := range path {
		if it == nil {
			return nil
		}
		switch p := part.(type) {
		case string:
			it = findChildByKey(it, p)
		case int:
			it = findChildByIndex(it, p)
		}
	}
	return it
}

func findChildByKey(n *jsonx.Node, key string) *jsonx.Node {
	if n.Kind != jsonx.Object {
		return nil
	}
	it := firstChild(n)
	for it != nil && it != n.End {
		if it.Key != "" {
			k := it.Key
			if unquoted, err := strconv.Unquote(it.Key); err == nil {
				k = unquoted
			}
			if k == key {
				return it
			}
		}
		it = nextSiblingNode(it)
	}
	return nil
}

func findChildByIndex(n *jsonx.Node, targetIdx int) *jsonx.Node {
	if n.Kind != jsonx.Array {
		return nil
	}
	it := firstChild(n)
	for i := 0; it != nil && it != n.End; i++ {
		if it.Index == targetIdx || i == targetIdx {
			return it
		}
		it = nextSiblingNode(it)
	}
	return nil
}

func nextSiblingNode(n *jsonx.Node) *jsonx.Node {
	if n.HasChildren() {
		return n.End.Next
	}
	if n.ChunkEnd != nil {
		return n.ChunkEnd.Next
	}
	return n.Next
}

func firstChild(n *jsonx.Node) *jsonx.Node {
	if n.IsCollapsed() {
		return n.Collapsed
	}
	return n.Next
}

func cloneSubtree(root *jsonx.Node) *jsonx.Node {
	if root == nil {
		return nil
	}
	end := root.End

	nodeMap := make(map[*jsonx.Node]*jsonx.Node)
	for it := root; it != nil; it = it.Next {
		cp := &jsonx.Node{
			Depth:      it.Depth,
			Kind:       it.Kind,
			Key:        it.Key,
			Value:      it.Value,
			Size:       it.Size,
			Comma:      it.Comma,
			Index:      it.Index,
			LineNumber: it.LineNumber,
		}
		nodeMap[it] = cp
		if it == end || end == nil {
			break
		}
	}

	for it := root; it != nil; it = it.Next {
		cp := nodeMap[it]
		if it.Prev != nil {
			cp.Prev = nodeMap[it.Prev]
		}
		if it.Next != nil && it != end && end != nil {
			cp.Next = nodeMap[it.Next]
		}
		if it.End != nil {
			cp.End = nodeMap[it.End]
		}
		if it.Parent != nil {
			cp.Parent = nodeMap[it.Parent]
		}
		if it.ChunkEnd != nil {
			cp.ChunkEnd = nodeMap[it.ChunkEnd]
		}
		if it == end || end == nil {
			break
		}
	}

	clonedRoot := nodeMap[root]
	clonedRoot.Prev = nil
	clonedRoot.Parent = nil
	return clonedRoot
}

func shiftDepth(root *jsonx.Node, delta int) {
	if delta == 0 {
		return
	}
	end := root.End
	for it := root; it != nil; it = it.Next {
		newDepth := int(it.Depth) + delta
		if newDepth < 0 {
			newDepth = 0
		}
		it.Depth = uint8(newDepth)
		if it == end || end == nil {
			break
		}
	}
}

func resetLineNumbers(root *jsonx.Node, startLine int) {
	curLine := startLine
	end := root.End
	for it := root; it != nil; it = it.Next {
		it.LineNumber = curLine
		curLine++
		if it == end || end == nil {
			break
		}
	}
}

func buildArray(elements []*jsonx.Node) *jsonx.Node {
	root := &jsonx.Node{
		Depth:      0,
		Kind:       jsonx.Array,
		Key:        "",
		Value:      "[",
		Size:       len(elements),
		Comma:      false,
		Index:      -1,
		LineNumber: 1,
	}
	end := &jsonx.Node{
		Depth:      0,
		Kind:       jsonx.Array,
		Key:        "",
		Value:      "]",
		Comma:      false,
		Index:      -1,
		LineNumber: 1,
		Parent:     root,
	}
	root.End = end

	if len(elements) == 0 {
		root.Next = end
		end.Prev = root
		return root
	}

	var prev *jsonx.Node = root
	curLine := 1

	for i, elem := range elements {
		cloned := cloneSubtree(elem)
		cloned.Key = ""
		cloned.Parent = root
		cloned.Index = i

		baseDepth := cloned.Depth
		shiftDepth(cloned, 1-int(baseDepth))

		isLast := (i == len(elements)-1)
		elemLast := cloned
		if cloned.End != nil {
			elemLast = cloned.End
			cloned.Comma = false
			elemLast.Comma = !isLast
		} else {
			cloned.Comma = !isLast
		}

		prev.Next = cloned
		cloned.Prev = prev

		for node := cloned; node != nil; node = node.Next {
			curLine++
			node.LineNumber = curLine
			if node == elemLast {
				break
			}
		}
		prev = elemLast
	}

	curLine++
	end.LineNumber = curLine
	prev.Next = end
	end.Prev = prev
	return root
}
