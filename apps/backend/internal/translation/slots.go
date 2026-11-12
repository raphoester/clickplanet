package translation

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// The page's lines are a subset of ICU MessageFormat, as src/i18n/message.ts reads them: {slot}, {n, plural, …},
// {n, selectordinal, …}, {v, select, …} and <tag>…</tag>. A translation must name the same slots and tags; it may
// write a count as a plain {n} where its language has one form for every number.
func SlotsDiffer(english, translated string) string {
	want, err := slotsOf(english)
	if err != nil {
		return "the English does not read: " + err.Error()
	}
	have, err := slotsOf(translated)
	if err != nil {
		return "it does not read: " + err.Error()
	}
	if slices.Equal(want, have) {
		return ""
	}
	return fmt.Sprintf("slots %s where the English has %s", strings.Join(have, " "), strings.Join(want, " "))
}

var (
	errUnclosed = errors.New("a brace or a tag is not closed")
	errNoOther  = errors.New(`a plural or a select has no "other" branch`)
	errArgument = errors.New("an argument does not read")
)

var (
	namePattern     = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*`)
	typePattern     = regexp.MustCompile(`^,\s*(plural|selectordinal|select)\s*,`)
	selectorPattern = regexp.MustCompile(`^\s*(=\d+|[A-Za-z_][A-Za-z0-9_]*)\s*\{`)
	tagPattern      = regexp.MustCompile(`^<([A-Za-z][A-Za-z0-9-]*)>`)
)

func slotsOf(text string) ([]string, error) {
	found := map[string]string{}
	rest, err := walk(text, "", found)
	if err != nil {
		return nil, err
	}
	if rest != "" {
		return nil, fmt.Errorf("%w: %q", errUnclosed, rest)
	}
	slots := make([]string, 0, len(found))
	for name, kind := range found {
		slots = append(slots, name+":"+kind)
	}
	slices.Sort(slots)
	return slots, nil
}

// Reads up to an unmatched "}" or the closing tag, and returns what is left after it.
func walk(text, closing string, found map[string]string) (string, error) {
	for text != "" {
		switch {
		case text[0] == '}':
			return text, nil
		case closing != "" && strings.HasPrefix(text, "</"+closing+">"):
			return text, nil
		case text[0] == '{':
			rest, err := argument(text[1:], found)
			if err != nil {
				return "", err
			}
			text = rest
		case tagPattern.MatchString(text):
			name := tagPattern.FindStringSubmatch(text)[1]
			found[name] = "tag"
			rest, err := walk(text[len(name)+2:], name, found)
			if err != nil {
				return "", err
			}
			if !strings.HasPrefix(rest, "</"+name+">") {
				return "", fmt.Errorf("%w: <%s>", errUnclosed, name)
			}
			text = rest[len(name)+3:]
		default:
			text = text[1:]
		}
	}
	return "", nil
}

func argument(text string, found map[string]string) (string, error) {
	name := namePattern.FindStringSubmatch(text)
	if name == nil {
		return "", fmt.Errorf("%w: {%s", errArgument, text)
	}
	text = text[len(name[0]):]
	if strings.HasPrefix(text, "}") {
		if _, ok := found[name[1]]; !ok {
			found[name[1]] = "value"
		}
		return text[1:], nil
	}
	kind := typePattern.FindStringSubmatch(text)
	if kind == nil {
		return "", fmt.Errorf("%w: {%s", errArgument, name[1])
	}
	text = text[len(kind[0]):]
	found[name[1]] = map[string]string{"plural": "value", "selectordinal": "ordinal", "select": "choice"}[kind[1]]
	other := false
	for {
		text = strings.TrimLeft(text, " \t\n")
		if strings.HasPrefix(text, "}") {
			break
		}
		selector := selectorPattern.FindStringSubmatch(text)
		if selector == nil {
			return "", fmt.Errorf("%w: {%s", errArgument, name[1])
		}
		other = other || selector[1] == "other"
		rest, err := walk(text[len(selector[0]):], "", found)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(rest, "}") {
			return "", fmt.Errorf("%w: {%s", errUnclosed, name[1])
		}
		text = rest[1:]
	}
	if !other {
		return "", fmt.Errorf("%w: {%s}", errNoOther, name[1])
	}
	return text[1:], nil
}
