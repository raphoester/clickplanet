// Package placeholders reads the `{path}` a card's words name a rule by, and fills it with the value a set plays. A
// path is a field of planet.v1.Rules as protojson names it, such as `clicks.tempo` or `bonuses.boxes.minInterval`, and
// only a number or a duration reads as a word in a sentence.
package placeholders

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

var (
	ErrInvalidPlaceholder = errors.New("not a placeholder a card may hold")
	ErrNoValue            = errors.New("the set plays no value at this path")
)

var placeholder = regexp.MustCompile(`\{([^{}]*)\}`)

var durationName = (&durationpb.Duration{}).ProtoReflect().Descriptor().FullName()

// Paths is every path a card may name, in the order planet.v1.Rules declares them.
func Paths() []string {
	var paths []string
	walk((&planetv1.Rules{}).ProtoReflect().Descriptor(), "", func(path string, _ protoreflect.FieldDescriptor) {
		paths = append(paths, path)
	})
	return paths
}

func walk(message protoreflect.MessageDescriptor, prefix string, visit func(path string, field protoreflect.FieldDescriptor)) {
	fields := message.Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		if field.IsList() || field.IsMap() {
			continue
		}
		path := prefix + field.JSONName()
		switch {
		case field.Kind() == protoreflect.MessageKind && field.Message().FullName() == durationName:
			visit(path, field)
		case field.Kind() == protoreflect.MessageKind:
			walk(field.Message(), path+".", visit)
		case number(field.Kind()):
			visit(path, field)
		}
	}
}

func number(kind protoreflect.Kind) bool {
	switch kind {
	case protoreflect.DoubleKind, protoreflect.FloatKind,
		protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return true
	default:
		return false
	}
}

// Check refuses words with a placeholder no path names, or a brace that opens or closes none.
func Check(words string) error {
	paths := Paths()
	var reasons []string
	for _, match := range placeholder.FindAllStringSubmatch(words, -1) {
		if !slices.Contains(paths, match[1]) {
			reasons = append(reasons, fmt.Sprintf("{%s} names no number or duration of the rules", match[1]))
		}
	}
	if strings.ContainsAny(placeholder.ReplaceAllString(words, ""), "{}") {
		reasons = append(reasons, "a brace opens or closes no placeholder")
	}
	if len(reasons) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidPlaceholder, strings.Join(reasons, "; "))
	}
	return nil
}

// Fill writes each placeholder's value as the set plays it: a number with no trailing zero, a duration in its largest
// whole unit.
func Fill(words string, played *planetv1.Rules) (string, error) {
	if err := Check(words); err != nil {
		return "", err
	}
	var failed error
	filled := placeholder.ReplaceAllStringFunc(words, func(found string) string {
		value, err := valueOf(played.ProtoReflect(), strings.Trim(found, "{}"))
		if err != nil {
			failed = errors.Join(failed, err)
		}
		return value
	})
	if failed != nil {
		return "", failed
	}
	return filled, nil
}

func valueOf(message protoreflect.Message, path string) (string, error) {
	segments := strings.Split(path, ".")
	for i, segment := range segments {
		field := message.Descriptor().Fields().ByJSONName(segment)
		if field == nil {
			return "", fmt.Errorf("%w: {%s}", ErrInvalidPlaceholder, path)
		}
		if !message.Has(field) {
			return "", fmt.Errorf("%w: {%s}", ErrNoValue, path)
		}
		value := message.Get(field)
		if i < len(segments)-1 {
			message = value.Message()
			continue
		}
		if field.Kind() == protoreflect.MessageKind {
			seconds := value.Message().Get(field.Message().Fields().ByName("seconds")).Int()
			nanos := value.Message().Get(field.Message().Fields().ByName("nanos")).Int()
			return durationWord(time.Duration(seconds)*time.Second + time.Duration(nanos)), nil
		}
		return numberWord(value, field.Kind()), nil
	}
	return "", fmt.Errorf("%w: {%s}", ErrInvalidPlaceholder, path)
}

func numberWord(value protoreflect.Value, kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(value.Uint(), 10)
	default:
		return strconv.FormatInt(value.Int(), 10)
	}
}

// "1 h", "2 min", "90 s", "500 ms": the largest unit that counts it whole.
func durationWord(d time.Duration) string {
	if d == 0 {
		return "0 s"
	}
	for _, unit := range []struct {
		size time.Duration
		name string
	}{{time.Hour, "h"}, {time.Minute, "min"}, {time.Second, "s"}, {time.Millisecond, "ms"}} {
		if d%unit.size == 0 {
			return fmt.Sprintf("%d %s", d/unit.size, unit.name)
		}
	}
	return d.String()
}
