package star

import (
	"bytes"
	gojson "encoding/json"
	"errors"
	"fmt"
	"strings"

	starjson "go.starlark.net/lib/json"
	"go.starlark.net/starlark"
)

// jsonNode is one value of a JSON text, with the exact text around its
// parts so unchanged parts can be written back byte for byte.
type jsonNode struct {
	start, end int // the value's span
	kind       byte
	// members of an object, or elements of an array (key is unused).
	items []jsonItem
	// post is the whitespace between the last item and the closing bracket.
	post string
}

// jsonItem is an object member or array element.
type jsonItem struct {
	pre   string // whitespace before the key (or element)
	key   string // decoded key
	ktext string // key as written, quotes included
	mid   string // from the end of the key to the value: ":" and spaces
	value *jsonNode
}

// jsonScanner parses JSON text into jsonNodes.
type jsonScanner struct {
	s   string
	pos int
}

var errJSONSyntax = errors.New("invalid JSON")

// ws skips whitespace and returns it.
func (p *jsonScanner) ws() string {
	start := p.pos
	for p.pos < len(p.s) && strings.IndexByte(" \t\r\n", p.s[p.pos]) >= 0 {
		p.pos++
	}
	return p.s[start:p.pos]
}

// str reads a string literal and returns it as written, quotes included.
func (p *jsonScanner) str() (string, error) {
	start := p.pos
	if p.pos >= len(p.s) || p.s[p.pos] != '"' {
		return "", errJSONSyntax
	}
	for p.pos++; p.pos < len(p.s); p.pos++ {
		switch p.s[p.pos] {
		case '\\':
			p.pos++
		case '"':
			p.pos++
			return p.s[start:p.pos], nil
		}
	}
	return "", errJSONSyntax
}

// value parses the value at the current position.
func (p *jsonScanner) value(depth int) (*jsonNode, error) {
	if depth > maxYAMLDepth {
		return nil, fmt.Errorf("value nested too deeply")
	}
	if p.pos >= len(p.s) {
		return nil, errJSONSyntax
	}
	n := &jsonNode{start: p.pos, kind: p.s[p.pos]}
	switch c := p.s[p.pos]; c {
	case '{', '[':
		closer := byte('}')
		if c == '[' {
			closer = ']'
		}
		p.pos++
		for {
			pre := p.ws()
			if p.pos < len(p.s) && p.s[p.pos] == closer && len(n.items) == 0 {
				n.post = pre
				break
			}
			it := jsonItem{pre: pre}
			if c == '{' {
				k, err := p.str()
				if err != nil {
					return nil, err
				}
				it.ktext = k
				if err := gojsonUnquote(k, &it.key); err != nil {
					return nil, err
				}
				mid := p.pos
				p.ws()
				if p.pos >= len(p.s) || p.s[p.pos] != ':' {
					return nil, errJSONSyntax
				}
				p.pos++
				p.ws()
				it.mid = p.s[mid:p.pos]
			}
			v, err := p.value(depth + 1)
			if err != nil {
				return nil, err
			}
			it.value = v
			n.items = append(n.items, it)
			post := p.ws()
			if p.pos >= len(p.s) {
				return nil, errJSONSyntax
			}
			if p.s[p.pos] == ',' {
				p.pos++
				continue
			}
			if p.s[p.pos] != closer {
				return nil, errJSONSyntax
			}
			n.post = post
			break
		}
		p.pos++
	case '"':
		if _, err := p.str(); err != nil {
			return nil, err
		}
	default:
		for p.pos < len(p.s) && strings.IndexByte(",]} \t\r\n", p.s[p.pos]) < 0 {
			p.pos++
		}
		if p.pos == n.start {
			return nil, errJSONSyntax
		}
	}
	n.end = p.pos
	return n, nil
}

// gojsonUnquote decodes a JSON string literal. Control characters are
// escaped first, since Starlark's decoder accepts them raw.
func gojsonUnquote(lit string, out *string) error {
	var b strings.Builder
	for _, r := range lit {
		if r < 0x20 {
			fmt.Fprintf(&b, "\\u%04x", r)
		} else {
			b.WriteRune(r)
		}
	}
	return gojson.Unmarshal([]byte(b.String()), out)
}

// jsonLike writes values in the layout of an original JSON text.
type jsonLike struct {
	thread *starlark.Thread
	src    string
	indent string
	// inlineLists is set when the original writes a list of plain values
	// on one line, so new lists of plain values are written that way too.
	inlineLists bool
	// spaced is set when the original puts a space after colons.
	spaced bool
}

// same reports whether v is the value the node holds, keys in the same
// order.
func (w *jsonLike) same(v starlark.Value, n *jsonNode) bool {
	decoded, err := starlark.Call(w.thread, starjson.Module.Members["decode"], starlark.Tuple{starlark.String(w.src[n.start:n.end])}, nil)
	if err != nil {
		return false
	}
	var a, b bytes.Buffer
	return encodeJSON(&a, v, 0) == nil && encodeJSON(&b, decoded, 0) == nil && bytes.Equal(a.Bytes(), b.Bytes())
}

// write writes v at depth, reusing n's text where v hasn't changed.
func (w *jsonLike) write(buf *bytes.Buffer, v starlark.Value, n *jsonNode, depth int) error {
	if n != nil && w.same(v, n) {
		buf.WriteString(w.src[n.start:n.end])
		return nil
	}
	// A one-line object or list of plain values in an indented file that
	// gains nested values is written out indented rather than on one line.
	if n != nil && (n.kind == '{' || n.kind == '[') && w.indent != "" && !w.multiline(n) && nodeFlat(n) && !flat(v) {
		return w.fresh(buf, v, depth)
	}
	switch t := v.(type) {
	case *starlark.Dict:
		if n == nil || n.kind != '{' {
			return w.fresh(buf, v, depth)
		}
		old := map[string]int{}
		for i, it := range n.items {
			old[it.key] = i
		}
		items := make([]jsonItem, 0, t.Len())
		for _, kv := range t.Items() {
			k, ok := starlark.AsString(kv[0])
			if !ok {
				return fmt.Errorf("dict key %s is not a string", kv[0])
			}
			var it jsonItem
			if at, ok := old[k]; ok {
				it = n.items[at]
				it.pre = w.pre(n, at, len(items), depth)
			} else {
				var kb bytes.Buffer
				writeJSONString(&kb, k)
				it = jsonItem{pre: w.newPre(n, depth), ktext: kb.String(), mid: w.newMid(n)}
				if len(items) == 0 {
					it.pre = w.pre(n, -1, 0, depth)
				}
			}
			it.key = k
			items = append(items, it)
		}
		buf.WriteByte('{')
		for i, it := range items {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(it.pre)
			buf.WriteString(it.ktext)
			buf.WriteString(it.mid)
			val, _, _ := t.Get(starlark.String(it.key))
			if err := w.write(buf, val, it.value, depth+1); err != nil {
				return err
			}
		}
		buf.WriteString(w.post(n, len(items), depth))
		buf.WriteByte('}')
		return nil
	case *starlark.List, starlark.Tuple:
		if n == nil || n.kind != '[' {
			return w.fresh(buf, v, depth)
		}
		buf.WriteByte('[')
		iter := starlark.Iterate(v)
		defer iter.Done()
		var x starlark.Value
		i := 0
		for ; iter.Next(&x); i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			var orig *jsonNode
			pre := w.newPre(n, depth)
			if i < len(n.items) {
				orig, pre = n.items[i].value, n.items[i].pre
			} else if i == 0 {
				pre = w.pre(n, -1, 0, depth)
			}
			buf.WriteString(pre)
			if err := w.write(buf, x, orig, depth+1); err != nil {
				return err
			}
		}
		buf.WriteString(w.post(n, i, depth))
		buf.WriteByte(']')
		return nil
	}
	return encodeJSON(buf, v, depth)
}

// multiline reports whether a container spans lines in the original.
func (w *jsonLike) multiline(n *jsonNode) bool {
	return strings.Contains(w.src[n.start:n.end], "\n")
}

// newPre is the whitespace before an item added to n: like its last item,
// or a new indented line when it had none and the file is indented.
func (w *jsonLike) newPre(n *jsonNode, depth int) string {
	if len(n.items) > 0 {
		pre := n.items[len(n.items)-1].pre
		if len(n.items) == 1 && !strings.Contains(pre, "\n") {
			if w.multiline(n) {
				return "\n" + strings.Repeat(w.indent, depth+1)
			}
			if pre == "" && w.spaced {
				// One item gives no separator to copy; follow the file's
				// "key": value spacing.
				return " "
			}
		}
		return pre
	}
	if w.indent == "" {
		return ""
	}
	return "\n" + strings.Repeat(w.indent, depth+1)
}

// pre is the whitespace before an item written at position pos that was
// at position at in n (-1 for a new item): the first position takes the
// first item's whitespace, and an item moved off it takes its neighbour's.
func (w *jsonLike) pre(n *jsonNode, at, pos, depth int) string {
	switch {
	case pos == 0 && len(n.items) > 0:
		return n.items[0].pre
	case pos == 0:
		return w.newPre(n, depth)
	case at > 0:
		return n.items[at].pre
	case len(n.items) > 1:
		return n.items[1].pre
	}
	return w.newPre(n, depth)
}

// newMid is the text between a new key and its value.
func (w *jsonLike) newMid(n *jsonNode) string {
	if len(n.items) > 0 {
		return n.items[len(n.items)-1].mid
	}
	if w.indent == "" {
		return ":"
	}
	return ": "
}

// post is the whitespace before the closing bracket of n with count items.
func (w *jsonLike) post(n *jsonNode, count, depth int) string {
	if count == 0 {
		return ""
	}
	if len(n.items) > 0 || w.indent == "" {
		return n.post
	}
	return "\n" + strings.Repeat(w.indent, depth)
}

// fresh writes a value that has no original text, in the file's style:
// indented for depth, with plain lists on one line when the file writes
// them that way, or on one line throughout for a compact file.
func (w *jsonLike) fresh(buf *bytes.Buffer, v starlark.Value, depth int) error {
	if depth > maxYAMLDepth {
		return fmt.Errorf("value nested too deeply")
	}
	comma, colon := ",", ":"
	if w.spaced {
		comma, colon = ", ", ": "
	}
	switch t := v.(type) {
	case *starlark.Dict:
		if t.Len() == 0 {
			buf.WriteString("{}")
			return nil
		}
		buf.WriteByte('{')
		for i, kv := range t.Items() {
			k, ok := starlark.AsString(kv[0])
			if !ok {
				return fmt.Errorf("dict key %s is not a string", kv[0])
			}
			if i > 0 {
				buf.WriteByte(',')
			}
			if w.indent != "" {
				buf.WriteString("\n" + strings.Repeat(w.indent, depth+1))
			} else if i > 0 {
				buf.WriteString(strings.TrimPrefix(comma, ","))
			}
			writeJSONString(buf, k)
			buf.WriteString(colon)
			if err := w.fresh(buf, kv[1], depth+1); err != nil {
				return err
			}
		}
		if w.indent != "" {
			buf.WriteString("\n" + strings.Repeat(w.indent, depth))
		}
		buf.WriteByte('}')
		return nil
	case *starlark.List, starlark.Tuple:
		seq := v.(starlark.Indexable)
		if seq.Len() == 0 {
			buf.WriteString("[]")
			return nil
		}
		inline := w.indent == "" || (w.inlineLists && plainList(v))
		buf.WriteByte('[')
		for i := range seq.Len() {
			if i > 0 {
				buf.WriteByte(',')
			}
			if !inline {
				buf.WriteString("\n" + strings.Repeat(w.indent, depth+1))
			} else if i > 0 {
				buf.WriteString(strings.TrimPrefix(comma, ","))
			}
			if err := w.fresh(buf, seq.Index(i), depth+1); err != nil {
				return err
			}
		}
		if !inline {
			buf.WriteString("\n" + strings.Repeat(w.indent, depth))
		}
		buf.WriteByte(']')
		return nil
	}
	return encodeJSON(buf, v, depth)
}

// flat reports whether v is a plain value, a plain list, or an object of
// plain values: something that reads well on one line.
func flat(v starlark.Value) bool {
	switch t := v.(type) {
	case *starlark.Dict:
		for _, kv := range t.Items() {
			switch kv[1].(type) {
			case *starlark.Dict, *starlark.List, starlark.Tuple:
				return false
			}
		}
		return true
	case *starlark.List, starlark.Tuple:
		return plainList(v) || v.(starlark.Indexable).Len() == 0
	}
	return true
}

// nodeFlat reports whether an object or list in the text holds only
// plain values.
func nodeFlat(n *jsonNode) bool {
	for _, it := range n.items {
		if it.value.kind == '{' || it.value.kind == '[' {
			return false
		}
	}
	return true
}

// plainList reports whether v is a non-empty list of strings, numbers,
// booleans or nulls.
func plainList(v starlark.Value) bool {
	seq, ok := v.(starlark.Indexable)
	if !ok || seq.Len() == 0 {
		return false
	}
	if _, isStr := v.(starlark.String); isStr {
		return false
	}
	for i := range seq.Len() {
		switch seq.Index(i).(type) {
		case *starlark.Dict, *starlark.List, starlark.Tuple:
			return false
		}
	}
	return true
}

// hasInlineList reports whether the text writes a non-empty list of plain
// values on one line.
func hasInlineList(src string, n *jsonNode) bool {
	if n.kind == '[' && len(n.items) > 0 && !strings.Contains(src[n.start:n.end], "\n") {
		plain := true
		for _, it := range n.items {
			if it.value.kind == '{' || it.value.kind == '[' {
				plain = false
			}
		}
		if plain {
			return true
		}
	}
	for _, it := range n.items {
		if hasInlineList(src, it.value) {
			return true
		}
	}
	return false
}

// jsonIndentUnit guesses the indentation a JSON text uses: the
// whitespace before its first indented key, or none for compact text.
func jsonIndentUnit(src string) string {
	i := strings.Index(src, "\n")
	for i >= 0 && i+1 < len(src) {
		rest := src[i+1:]
		trimmed := strings.TrimLeft(rest, " \t")
		if lead := rest[:len(rest)-len(trimmed)]; lead != "" && trimmed != "" && trimmed[0] != '\n' {
			return lead
		}
		next := strings.Index(rest, "\n")
		if next < 0 {
			break
		}
		i += 1 + next
	}
	if strings.Contains(strings.TrimSpace(src), "\n") {
		return "  "
	}
	return ""
}

// encodeLike encodes v in the layout of the JSON text src: unchanged
// values keep their exact text, and changed or new ones follow the
// separators and indentation around them.
func encodeLike(thread *starlark.Thread, v starlark.Value, src string) (string, error) {
	p := &jsonScanner{s: src}
	lead := p.ws()
	root, err := p.value(0)
	if err != nil {
		return "", err
	}
	trail := p.ws()
	if p.pos != len(src) {
		return "", errJSONSyntax
	}
	w := &jsonLike{thread: thread, src: src, indent: jsonIndentUnit(src), inlineLists: hasInlineList(src, root), spaced: strings.Contains(src, ": ")}
	if len(root.items) == 0 {
		// An empty file has no layout to follow; write it like a new one.
		w.indent, w.spaced = "  ", true
	}
	var buf bytes.Buffer
	buf.WriteString(lead)
	if err := w.write(&buf, v, root, 0); err != nil {
		return "", err
	}
	buf.WriteString(trail)
	return buf.String(), nil
}

// jsonEncodeLike is json.encode_like(x, original).
func jsonEncodeLike(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var x starlark.Value
	var original string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "x", &x, "original", &original); err != nil {
		return nil, err
	}
	out, err := encodeLike(thread, x, original)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", b.Name(), err)
	}
	return starlark.String(out), nil
}
