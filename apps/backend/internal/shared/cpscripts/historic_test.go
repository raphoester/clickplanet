package cpscripts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpscripts"
)

func TestTheScriptsNoOneWritesTodayAreHistoric(t *testing.T) {
	for _, r := range []rune{'𒁜', '𒂂', '𒐫', '𒑰', '𓀀', 'ᚠ', 'ⲁ', '𐌰', '𐤀', '𐀀', '𐘀'} {
		assert.True(t, cpscripts.Historic(r), "%q %U", r, r)
	}
}

func TestLivingAlphabetsEmojisAndPunctuationAreNot(t *testing.T) {
	for _, r := range []rune{
		'a', 'Z', 'é', 'ß', 'Ж', 'Ω', 'م', 'ـ', 'ש', 'न', 'ก', '東', 'タ', '김', 'ㅋ', 'ⵣ', 'ꀀ', 'Ａ',
		'0', '٣', ' ', '!', '?', '_', '€', '🌍', '😂', '🫩', '‍', '️', '\U0001f3fb', '\U000e0067',
	} {
		assert.False(t, cpscripts.Historic(r), "%q %U", r, r)
	}
}
