// Package placeholders reads the `{path}` a card's words name a value by, and fills it with the value a battle plays. A
// path is a field of planet.v1.Rules as protojson names it, such as `clicks.tempo` or `charges.bomb.rings`, or one of
// bonuses.v1.Rules under `bonuses.`, such as `bonuses.boxes.minInterval`, and only a number or a duration reads as a
// word in a sentence; or a value of the battle, such as `battle.points.first`.
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

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

var (
	ErrInvalidPlaceholder = errors.New("not a placeholder a card may hold")
	ErrNoValue            = errors.New("the set plays no value at this path")
)

var placeholder = regexp.MustCompile(`\{([^{}]*)\}`)

var durationName = (&durationpb.Duration{}).ProtoReflect().Descriptor().FullName()

// offers is where the offers' values are named: the place they held in planet's rules until they were a module of their
// own, so a card that names one reads as it did.
const offers = "bonuses."

// Played is what a set plays: planet's values, and the offers' of the set of the same name. Offers is nil when the
// offers keep no set of that name: a card names none of their values.
type Played struct {
	Planet *planetv1.Rules
	Offers *bonusesv1.Rules
}

// Battle is what a card may name of the battle that plays its set: its points' times, and what each rank scores in
// it, by rank.
type Battle struct {
	Times  uint32
	Points map[uint32]uint32
}

const battleTimes = "battle.times"

// The ranks whose points a card may name, the first's first.
var places = []string{"first", "second", "third"}

// BattlePaths is every path a card may name of the battle.
func BattlePaths() []string {
	paths := make([]string, 0, 1+len(places))
	paths = append(paths, battleTimes)
	for _, place := range places {
		paths = append(paths, "battle.points."+place)
	}
	return paths
}

func (b Battle) valueOf(path string) (string, error) {
	if path == battleTimes {
		return strconv.FormatUint(uint64(b.Times), 10), nil
	}
	place := slices.Index(places, strings.TrimPrefix(path, "battle.points."))
	if place < 0 {
		return "", fmt.Errorf("%w: {%s}", ErrInvalidPlaceholder, path)
	}
	points, ok := b.Points[uint32(place+1)] //nolint:gosec // three places.
	if !ok {
		return "", fmt.Errorf("%w: {%s}", ErrNoValue, path)
	}
	return strconv.FormatUint(uint64(points), 10), nil
}

// RulePaths is every path a card may name of the rules: planet's in the order planet.v1.Rules declares them, then the
// offers' in bonuses.v1.Rules' order.
func RulePaths() []string {
	var paths []string
	visit := func(path string, _ protoreflect.FieldDescriptor) { paths = append(paths, path) }
	walk((&planetv1.Rules{}).ProtoReflect().Descriptor(), "", visit)
	walk((&bonusesv1.Rules{}).ProtoReflect().Descriptor(), offers, visit)
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
	paths := slices.Concat(RulePaths(), BattlePaths())
	var reasons []string
	for _, match := range placeholder.FindAllStringSubmatch(words, -1) {
		if !slices.Contains(paths, match[1]) {
			reasons = append(reasons, fmt.Sprintf("{%s} names no number or duration of the rules or the battle", match[1]))
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

// Fill writes each placeholder's value as the battle and its set play it: a number with no trailing zero, a duration
// in its largest whole unit.
func Fill(words string, played Played, battle Battle) (string, error) {
	if err := Check(words); err != nil {
		return "", err
	}
	battlePaths := BattlePaths()
	var failed error
	filled := placeholder.ReplaceAllStringFunc(words, func(found string) string {
		path := strings.Trim(found, "{}")
		var (
			value string
			err   error
		)
		switch {
		case slices.Contains(battlePaths, path):
			value, err = battle.valueOf(path)
		case strings.HasPrefix(path, offers):
			value, err = played.offersValueOf(path)
		default:
			value, err = valueOf(played.Planet.ProtoReflect(), path, path)
		}
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

func (p Played) offersValueOf(path string) (string, error) {
	if p.Offers == nil {
		return "", fmt.Errorf("%w: {%s}", ErrNoValue, path)
	}
	return valueOf(p.Offers.ProtoReflect(), strings.TrimPrefix(path, offers), path)
}

// valueOf reads the value at path in the message; named is the path the card wrote, which every refusal quotes.
func valueOf(message protoreflect.Message, path, named string) (string, error) {
	segments := strings.Split(path, ".")
	for i, segment := range segments {
		field := message.Descriptor().Fields().ByJSONName(segment)
		if field == nil {
			return "", fmt.Errorf("%w: {%s}", ErrInvalidPlaceholder, named)
		}
		if !message.Has(field) {
			return "", fmt.Errorf("%w: {%s}", ErrNoValue, named)
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
	return "", fmt.Errorf("%w: {%s}", ErrInvalidPlaceholder, named)
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
