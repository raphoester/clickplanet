package rotator_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rotator"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	address = "203.0.113.7"
	line    = "2001:db8:1:2::/64"

	slotSeconds = 300
)

func production() rotator.Rules {
	return rotator.Rules{
		Window:      24 * time.Hour,
		MinSeconds:  15,
		MinByChance: 90,
		MaxShare:    0.05,

		MinSlots:        24,
		CertainSlots:    48,
		CertainByChance: 300,
	}
}

type harness struct {
	watchdog *rotator.Watchdog
	clock    *cptime.FixedClock
}

func newHarness(rules rotator.Rules) *harness {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC))
	watchdog := rotator.New(rotator.Config{Slot: 5 * time.Minute}, func() rotator.Rules { return rules }, clock, func(float64) {})
	return &harness{watchdog: watchdog, clock: clock}
}

type player struct {
	scope    string
	account  string
	flag     string
	signedIn bool
}

func linked(scope, account, flag string) player {
	return player{scope: scope, account: account, flag: flag, signedIn: true}
}

func guest(scope, account, flag string) player {
	return player{scope: scope, account: account, flag: flag}
}

func (h *harness) clickOf(p player) detect.Click {
	return detect.Click{Scope: p.scope, Account: p.account, SignedIn: p.signedIn, Tile: 1, Country: p.flag, At: h.clock.Now()}
}

func (h *harness) click(p player) detect.Verdict {
	click := h.clickOf(p)
	h.watchdog.Attempted(click)
	verdict, _ := h.watchdog.Watch(click)
	return verdict
}

type reading struct {
	verdict  detect.Verdict
	evidence detect.Evidence
}

func (h *harness) read(p player) reading {
	verdict, evidence := h.watchdog.Watch(h.clickOf(p))
	return reading{verdict: verdict, evidence: evidence}
}

type turn struct {
	who      player
	from, to int
}

func (h *harness) slot(turns ...turn) {
	for second := range slotSeconds {
		for _, turn := range turns {
			if second >= turn.from && second < turn.to {
				h.click(turn.who)
			}
		}
		h.clock.Advance(time.Second)
	}
}

func (h *harness) turns(slots int, players ...player) {
	each := slotSeconds / len(players)

	for range slots {
		turns := make([]turn, 0, len(players))
		for i, p := range players {
			turns = append(turns, turn{who: p, from: i * each, to: i*each + 37})
		}
		h.slot(turns...)
	}
}

type person struct {
	player
	rng      *rand.Rand
	burst    [2]int
	pause    [2]int
	clicking bool
	until    int
	away     bool
}

func (p *person) clicks(second int) bool {
	if second >= p.until {
		p.clicking = !p.clicking
		span := p.pause
		if p.clicking {
			span = p.burst
		}
		p.until = second + span[0] + p.rng.IntN(span[1]-span[0]+1)
	}
	return p.clicking
}

type crowd struct {
	strongest detect.Verdict
	any       int
	several   int
}

func (c *crowd) shared() float64 { return float64(c.several) / float64(c.any) }

func (h *harness) second(c *crowd, clicking []player) {
	for _, p := range clicking {
		c.strongest = max(c.strongest, h.click(p))
	}
	if len(clicking) > 0 {
		c.any++
	}
	if len(clicking) > 1 {
		c.several++
	}
	h.clock.Advance(time.Second)
}

func TestThreeAccountsThatTakeTurnsAreRead(t *testing.T) {
	h := newHarness(production())

	one := linked(address, "desktop-one", "pl")
	two := linked(address, "desktop-two", "pl")
	phone := linked(address, "phone", "pl")

	const (
		slots  = 118
		shared = 101
	)

	for i := range slots {
		switch {
		case i%7 == 0:
			h.slot(turn{one, 0, 29})
		case i%40 == 1:
			h.slot(turn{one, 0, 37}, turn{two, 36, 73}, turn{phone, 200, 237})
		default:
			h.slot(turn{one, 0, 37}, turn{two, 100, 137}, turn{phone, 200, 237})
		}
	}

	for p, accomplices := range map[player][]detect.Accomplice{
		one:   {{Account: "desktop-two", SignedIn: true, Scope: address}, {Account: "phone", SignedIn: true, Scope: address}},
		two:   {{Account: "desktop-one", SignedIn: true, Scope: address}, {Account: "phone", SignedIn: true, Scope: address}},
		phone: {{Account: "desktop-one", SignedIn: true, Scope: address}, {Account: "desktop-two", SignedIn: true, Scope: address}},
	} {
		r := h.read(p)

		require.Equal(t, detect.Certain, r.verdict, p.account)
		assert.Equal(t, "rotation", r.evidence.Rule)
		assert.True(t, r.evidence.Turns, p.account)
		assert.ElementsMatch(t, []detect.Field{
			{Key: "accounts", Value: 3},
			{Key: "flag", Value: "pl"},
			{Key: "sharedSlots", Value: shared},
			{Key: "sharedSeconds", Value: 3},
			{Key: "byChance", Value: float64(1383)},
		}, r.evidence.Fields, p.account)
		assert.Equal(t, accomplices, r.evidence.Accomplices, p.account)
	}

	assert.Equal(t, "rotation accounts=3 byChance=1383 flag=pl sharedSeconds=3 sharedSlots=101", h.read(one).evidence.String())
}

func TestAHouseholdOfTwoIsClear(t *testing.T) {
	h := newHarness(production())

	//nolint:gosec // seeded test PRNG
	rng := rand.New(rand.NewPCG(10, 10))

	household := []*person{
		{player: linked(line, "parent", "fr"), rng: rng, burst: [2]int{5, 25}, pause: [2]int{30, 106}},
		{player: linked(line, "child", "fr"), rng: rng, burst: [2]int{5, 25}, pause: [2]int{30, 106}},
	}

	var seen crowd
	for second := range 6 * 3600 {
		var clicking []player
		for _, p := range household {
			if p.clicks(second) {
				clicking = append(clicking, p.player)
			}
		}
		h.second(&seen, clicking)
	}

	assert.Equal(t, detect.Clear, seen.strongest)
	assert.InDelta(t, 0.10, seen.shared(), 0.03, "two people share about a tenth of their seconds")
}

type school struct {
	rng      *rand.Rand
	students []*person
}

func newSchool(rng *rand.Rand, students int) *school {
	s := &school{rng: rng}
	for i := range students {
		s.students = append(s.students, &person{
			player: linked(address, fmt.Sprintf("student-%d", i), "pt"),
			rng:    rng,
			burst:  [2]int{4, 12},
			pause:  [2]int{60, 140},
			until:  rng.IntN(140),
		})
	}
	return s
}

func (s *school) clicking(second int) []player {
	var clicking []player
	for _, student := range s.students {
		if second%600 == 0 {
			student.away = s.rng.IntN(7) != 0
		}
		if student.clicks(second) && !student.away {
			clicking = append(clicking, student.player)
		}
	}
	return clicking
}

func TestASchoolBehindOneAddressIsClear(t *testing.T) {
	h := newHarness(production())

	//nolint:gosec // seeded test PRNG
	rng := rand.New(rand.NewPCG(10, 8))

	school := newSchool(rng, 16)

	var seen crowd
	for second := range 10 * 3600 {
		h.second(&seen, school.clicking(second))
	}

	assert.Equal(t, detect.Clear, seen.strongest)
	assert.InDelta(t, 0.07, seen.shared(), 0.03, "a school shares about 7% of its seconds")
}

func TestTurnsInsideASchoolAreReadAndTheSchoolIsNot(t *testing.T) {
	h := newHarness(production())

	//nolint:gosec // seeded test PRNG
	rng := rand.New(rand.NewPCG(10, 9))

	school := newSchool(rng, 16)
	trio := []player{linked(address, "desktop-one", "pt"), linked(address, "desktop-two", "pt"), linked(address, "phone", "pt")}

	var seen crowd
	for second := range 5 * 3600 {
		if at := second % slotSeconds; at%100 < 37 {
			h.click(trio[at/100])
		}
		h.second(&seen, school.clicking(second))
	}

	assert.Equal(t, detect.Clear, seen.strongest, "no student is read, alone or with the trio")

	for _, p := range trio {
		r := h.read(p)
		require.Equal(t, detect.Suspect, r.verdict, p.account)
		assert.Contains(t, r.evidence.Fields, detect.Field{Key: "accounts", Value: 3}, p.account)
		assert.Len(t, r.evidence.Accomplices, 2, p.account)
	}
}

func TestOneAccountAloneIsClear(t *testing.T) {
	h := newHarness(production())

	alone := linked(line, "alone", "it")

	for range 60 {
		h.slot(turn{alone, 0, 120}, turn{alone, 200, 260})
	}

	assert.Equal(t, detect.Clear, h.read(alone).verdict)
}

func TestUnderTwoHoursOfTurnsIsNotEnough(t *testing.T) {
	h := newHarness(production())

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")

	h.turns(23, one, two)
	assert.Equal(t, detect.Clear, h.read(one).verdict)
	assert.Equal(t, detect.Clear, h.read(two).verdict)

	h.turns(1, one, two)
	assert.Equal(t, detect.Suspect, h.read(one).verdict)
	assert.Equal(t, detect.Suspect, h.read(two).verdict)
}

func TestTurnsBetweenASignedInAccountAndAGuestTabAreRead(t *testing.T) {
	h := newHarness(production())

	account, tab := linked(line, "account", "hu"), guest(line, "tab", "hu")

	h.turns(30, account, tab)

	r := h.read(account)
	require.Equal(t, detect.Suspect, r.verdict)
	assert.Equal(t, []detect.Accomplice{{Account: "tab", Scope: line}}, r.evidence.Accomplices, "where the guest clicked last")
	assert.Equal(t, detect.Suspect, h.read(tab).verdict)
}

func TestTwoGuestsOnOneBankAreNeverPaired(t *testing.T) {
	h := newHarness(production())

	one, two := guest(line, "one", "hu"), guest(line, "two", "hu")

	h.turns(60, one, two)

	assert.Equal(t, detect.Clear, h.read(one).verdict)
	assert.Equal(t, detect.Clear, h.read(two).verdict)
}

func TestAccountsOnOtherFlagsAreNoRotation(t *testing.T) {
	h := newHarness(production())

	one, two := linked(line, "one", "hu"), linked(line, "two", "ro")

	h.turns(60, one, two)

	assert.Equal(t, detect.Clear, h.read(one).verdict)
	assert.Equal(t, detect.Clear, h.read(two).verdict)
}

func TestOneHandOnTwoNetworksIsRead(t *testing.T) {
	h := newHarness(production())

	account := linked(address, "account", "ru")
	var tab player
	for i := range 50 {
		tab = guest(fmt.Sprintf("2001:db8:%x::/64", i), "tab", "ru")
		h.slot(turn{account, 0, 60}, turn{tab, 150, 210})
	}

	r := h.read(account)
	require.Equal(t, detect.Certain, r.verdict, "a guest on a VPN's exits, a new /64 each slot")
	assert.Equal(t, []detect.Accomplice{{Account: "tab", Scope: tab.scope}}, r.evidence.Accomplices)
	assert.Equal(t, detect.Certain, h.read(tab).verdict)
}

func TestTwoPeopleOfOneFlagOnTwoNetworksAreClear(t *testing.T) {
	h := newHarness(production())

	//nolint:gosec // seeded test PRNG
	rng := rand.New(rand.NewPCG(11, 3))

	apart := []*person{
		{player: linked(line, "here", "ru"), rng: rng, burst: [2]int{20, 60}, pause: [2]int{10, 40}},
		{player: linked(address, "there", "ru"), rng: rng, burst: [2]int{20, 60}, pause: [2]int{10, 40}},
	}

	var seen crowd
	for second := range 12 * 3600 {
		var clicking []player
		for _, p := range apart {
			if p.clicks(second) {
				clicking = append(clicking, p.player)
			}
		}
		h.second(&seen, clicking)
	}

	assert.Equal(t, detect.Clear, seen.strongest, "they share what chance gives, however long they play")
}

func TestTurnsOlderThanTheWindowNoLongerCount(t *testing.T) {
	h := newHarness(production())

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")

	h.turns(30, one, two)
	require.Equal(t, detect.Suspect, h.read(one).verdict)

	h.clock.Advance(24 * time.Hour)
	h.turns(1, one, two)

	assert.Equal(t, detect.Clear, h.read(one).verdict)
}

func TestWithNoCertainSlotsItOnlyCorroborates(t *testing.T) {
	unset := production()
	unset.CertainSlots = 0
	h := newHarness(unset)

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")

	h.turns(200, one, two)

	assert.Equal(t, detect.Suspect, h.read(one).verdict)
}

func TestFourHoursOfTurnsReadCertain(t *testing.T) {
	h := newHarness(production())

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")

	for range 47 {
		h.slot(turn{one, 0, 60}, turn{two, 150, 210})
	}
	assert.Equal(t, detect.Suspect, h.read(one).verdict)

	h.slot(turn{one, 0, 60}, turn{two, 150, 210})
	assert.Equal(t, detect.Certain, h.read(one).verdict)
	assert.Equal(t, detect.Certain, h.read(two).verdict)
}

func TestSlowTurnsAreCertainOnlyOnceChanceIsSure(t *testing.T) {
	h := newHarness(production())

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")

	h.turns(65, one, two)
	assert.Equal(t, detect.Suspect, h.read(one).verdict, "37 seconds each: 65 slots are 296 seconds by chance")

	h.turns(1, one, two)
	assert.Equal(t, detect.Certain, h.read(one).verdict)
}

func TestACertainReadingNamesOnlyTheAccountsItIsSureOf(t *testing.T) {
	h := newHarness(production())

	one, two, three := linked(line, "one", "hu"), linked(line, "two", "hu"), linked(line, "three", "hu")

	for i := range 48 {
		turns := []turn{{one, 0, 60}, {two, 150, 210}}
		if i < 30 {
			turns = append(turns, turn{three, 240, 280})
		}
		h.slot(turns...)
	}

	r := h.read(one)
	require.Equal(t, detect.Certain, r.verdict)
	assert.Equal(t, []detect.Accomplice{{Account: "two", SignedIn: true, Scope: line}}, r.evidence.Accomplices)
	assert.Contains(t, r.evidence.Fields, detect.Field{Key: "accounts", Value: 2})
	assert.Equal(t, detect.Suspect, h.read(three).verdict, "thirty slots of turns with one")
}

func TestUnsetBoundsNeverRead(t *testing.T) {
	unset := production()
	unset.MinSlots, unset.CertainSlots = 0, 0
	h := newHarness(unset)

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")

	h.turns(200, one, two)

	assert.Equal(t, detect.Clear, h.read(one).verdict)
}
