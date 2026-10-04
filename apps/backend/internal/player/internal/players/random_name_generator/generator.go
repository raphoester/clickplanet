package random_name_generator

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var (
	Adjectives = []string{
		"Brave", "Swift", "Bold", "Iron", "Red", "Wild", "Lucky", "Silent", "Golden", "Mighty",
		"Rapid", "Fierce", "Clever", "Steady", "Royal", "Rogue", "Lone", "Proud", "Grand", "Stormy",
		"Sly", "Quick", "Fiery", "Frosty", "Noble", "Calm", "Daring", "Hidden", "Jolly", "Keen",
		"Nimble", "Rusty", "Sneaky", "Sunny", "Tiny", "Wise", "Happy", "Silver", "Dusty", "Blue",
	}
	Animals = []string{
		"Fox", "Wolf", "Bear", "Hawk", "Falcon", "Tiger", "Lion", "Eagle", "Raven", "Viper",
		"Otter", "Panda", "Shark", "Cobra", "Bison", "Moose", "Lynx", "Puma", "Jaguar", "Badger",
		"Beaver", "Coyote", "Gecko", "Heron", "Koala", "Lemur", "Mantis", "Orca", "Owl", "Parrot",
		"Penguin", "Python", "Rhino", "Walrus", "Yak", "Zebra", "Dingo", "Ferret", "Goose", "Hornet",
	}
)

const (
	lowestNumber  = 10
	highestNumber = 99
)

type Generator struct{}

var _ players.NameGenerator = Generator{}

func (Generator) NewName() (players.Name, error) {
	adjective, err := draw(len(Adjectives))
	if err != nil {
		return "", err
	}
	animal, err := draw(len(Animals))
	if err != nil {
		return "", err
	}
	number, err := draw(highestNumber - lowestNumber + 1)
	if err != nil {
		return "", err
	}

	return players.NameOf(fmt.Sprintf("%s%s%d", Adjectives[adjective], Animals[animal], lowestNumber+number)) //nolint:wrapcheck // every word passes the rule, which the tests pin.
}

func draw(n int) (int, error) {
	drawn, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, fmt.Errorf("failed to read random bytes: %w", err)
	}
	return int(drawn.Int64()), nil
}
