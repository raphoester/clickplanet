package random_name_generator_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/random_name_generator"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func TestADrawnNameIsAValidUsername(t *testing.T) {
	for range 100 {
		name, err := random_name_generator.Generator{}.NewName()
		require.NoError(t, err)

		valid, err := players.NameOf(string(name))
		require.NoError(t, err)
		assert.Equal(t, name, valid)
	}
}

func TestADrawnNameIsAnAdjectiveAnAnimalAndTwoDigits(t *testing.T) {
	for range 100 {
		name, err := random_name_generator.Generator{}.NewName()
		require.NoError(t, err)

		assert.Regexp(t, `^[A-Z][a-z]+[A-Z][a-z]+[0-9]{2}$`, string(name))
	}
	for _, word := range append(append([]string{}, random_name_generator.Adjectives...), random_name_generator.Animals...) {
		assert.Regexp(t, `^[A-Z][a-z]+$`, word, "so that every draw has the shape")
	}
}

func TestEveryWordAndNumberMakesAValidUsername(t *testing.T) {
	for _, adjective := range random_name_generator.Adjectives {
		for _, animal := range random_name_generator.Animals {
			for _, number := range []int{10, 99} {
				_, err := players.NameOf(fmt.Sprintf("%s%s%d", adjective, animal, number))
				require.NoError(t, err, "%s%s%d", adjective, animal, number)
			}
		}
	}
}

func TestNoWordIsListedTwice(t *testing.T) {
	words := append(append([]string{}, random_name_generator.Adjectives...), random_name_generator.Animals...)

	assert.Equal(t, len(words), cpcolls.NewSet(words...).Len())
}

func TestTwoNamesDiffer(t *testing.T) {
	first, err := random_name_generator.Generator{}.NewName()
	require.NoError(t, err)
	second, err := random_name_generator.Generator{}.NewName()
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}
