package cpflags

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	flagdata "github.com/raphoester/clickplanet.lol-backend/generated/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// BuiltInFlag is a flag the game ships with: its code, and its name in English.
type BuiltInFlag struct {
	Code string
	Name string
}

var (
	builtIn = sync.OnceValue(func() []BuiltInFlag {
		flags, err := builtInOf(flagdata.Countries())
		if err != nil {
			panic(fmt.Errorf("the flags the game ships with: %w", err))
		}
		return flags
	})
	builtInCodes = sync.OnceValue(func() *cpcolls.Set[string] {
		codes := cpcolls.NewSet[string]()
		for _, flag := range builtIn() {
			codes.Add(flag.Code)
		}
		return codes
	})
)

// BuiltIn is every flag the game ships with, in the page's order.
func BuiltIn() []BuiltInFlag {
	return slices.Clone(builtIn())
}

func IsBuiltIn(code string) bool {
	return builtInCodes().Contains(code)
}

var errNotAnObject = errors.New("not an object of codes to names")

// The file is one object from each code to its name: its order is the page's, which a map would lose.
func builtInOf(file []byte) ([]BuiltInFlag, error) {
	decoder := json.NewDecoder(bytes.NewReader(file))
	if open, err := decoder.Token(); err != nil || open != json.Delim('{') {
		return nil, errNotAnObject
	}
	var flags []BuiltInFlag
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to read a code: %w", err)
		}
		code, ok := key.(string)
		if !ok || code == "" {
			return nil, errNotAnObject
		}
		var name string
		if err := decoder.Decode(&name); err != nil || name == "" {
			return nil, fmt.Errorf("the flag %q has no name", code)
		}
		flags = append(flags, BuiltInFlag{Code: code, Name: name})
	}
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("failed to read the end of the flags: %w", err)
	}
	return flags, nil
}
