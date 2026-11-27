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
		Span:        time.Minute,
		MinSeconds:  3,
		MinByChance: 15,
		MaxShare:    0.05,

		MinSpans:        10,
		CertainSpans:    10,
		CertainByChance: 30,
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

// Each account clicks 37 seconds, and the next starts 3 seconds after: one shared span a slot for two accounts.
func (h *harness) turns(slots int, players ...player) {
	for range slots {
		turns := make([]turn, 0, len(players))
		for i, p := range players {
			turns = append(turns, turn{who: p, from: i * 40, to: i*40 + 37})
		}
		h.slot(turns...)
	}
}

// One hand plays each account in turn, for a minute or two, then the next 2 seconds after. It answers how long the
// play took to read certain, or -1.
func (h *harness) switching(rng *rand.Rand, lasting time.Duration, accounts ...player) time.Duration {
	start := h.clock.Now()
	certainAfter := time.Duration(-1)

	for i := 0; h.clock.Now().Sub(start) < lasting; i++ {
		for range 60 + rng.IntN(61) {
			if h.click(accounts[i%len(accounts)]) == detect.Certain && certainAfter < 0 {
				certainAfter = h.clock.Now().Sub(start)
			}
			h.clock.Advance(time.Second)
		}
		h.clock.Advance(2 * time.Second)
	}

	return certainAfter
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
			h.slot(turn{one, 0, 37}, turn{two, 36, 73}, turn{phone, 80, 117}, turn{one, 120, 157}, turn{phone, 160, 197}, turn{two, 200, 237})
		default:
			h.slot(turn{one, 0, 37}, turn{two, 40, 77}, turn{phone, 80, 117}, turn{one, 120, 157}, turn{phone, 160, 197}, turn{two, 200, 237})
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
			{Key: "sharedSpans", Value: 4 * shared},
			{Key: "sharedSeconds", Value: 3},
			{Key: "byChance", Value: float64(4609)},
		}, r.evidence.Fields, p.account)
		assert.Equal(t, accomplices, r.evidence.Accomplices, p.account)
	}

	assert.Equal(t, "rotation accounts=3 byChance=4609 flag=pl sharedSeconds=3 sharedSpans=404", h.read(one).evidence.String())
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

	cycle := []int{0, 1, 2, 0, 2, 1}

	var seen crowd
	for second := range 5 * 3600 {
		if at := second % slotSeconds; at < 240 && at%40 < 37 {
			h.click(trio[cycle[at/40]])
		}
		h.second(&seen, school.clicking(second))
	}

	assert.Equal(t, detect.Clear, seen.strongest, "no student is read, alone or with the trio")

	for _, p := range trio {
		r := h.read(p)
		require.Equal(t, detect.Certain, r.verdict, p.account)
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

func TestUnderTenSharedSpansIsNotEnough(t *testing.T) {
	h := newHarness(production())

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")

	h.turns(9, one, two)
	assert.Equal(t, detect.Clear, h.read(one).verdict, "nine spans, though chance gives 111 seconds")
	assert.Equal(t, detect.Clear, h.read(two).verdict)

	h.turns(1, one, two)
	assert.Equal(t, detect.Certain, h.read(one).verdict)
	assert.Equal(t, detect.Certain, h.read(two).verdict)
}

func TestTurnsBetweenASignedInAccountAndAGuestTabAreRead(t *testing.T) {
	h := newHarness(production())

	account, tab := linked(line, "account", "hu"), guest(line, "tab", "hu")

	h.turns(30, account, tab)

	r := h.read(account)
	require.Equal(t, detect.Certain, r.verdict)
	assert.Equal(t, []detect.Accomplice{{Account: "tab", Scope: line}}, r.evidence.Accomplices, "where the guest clicked last")
	assert.Equal(t, detect.Certain, h.read(tab).verdict)
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
		h.slot(turn{account, 20, 80}, turn{tab, 82, 142})
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
	require.Equal(t, detect.Certain, h.read(one).verdict)

	h.clock.Advance(24 * time.Hour)
	h.turns(1, one, two)

	assert.Equal(t, detect.Clear, h.read(one).verdict)
}

func TestWithNoCertainSpansItOnlyCorroborates(t *testing.T) {
	unset := production()
	unset.CertainSpans = 0
	h := newHarness(unset)

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")

	h.turns(200, one, two)

	assert.Equal(t, detect.Suspect, h.read(one).verdict)
}

func TestOneHandSwitchingEveryMinuteOrTwoIsCertainWithinHalfAnHour(t *testing.T) {
	h := newHarness(production())

	//nolint:gosec // seeded test PRNG
	rng := rand.New(rand.NewPCG(26, 11))

	main, alt := linked(address, "main", "tr"), guest(address, "alt", "tr")

	certainAfter := h.switching(rng, time.Hour, main, alt)

	require.Positive(t, certainAfter, "never read certain")
	assert.Less(t, certainAfter, 30*time.Minute)
	assert.Greater(t, certainAfter, 9*time.Minute, "ten switches at least, each turn a minute or more")
}

func TestTwoPeopleAtAHandsPaceAreClear(t *testing.T) {
	h := newHarness(production())

	//nolint:gosec // seeded test PRNG
	rng := rand.New(rand.NewPCG(26, 12))

	pair := []*person{
		{player: linked(address, "first", "tr"), rng: rng, burst: [2]int{60, 120}, pause: [2]int{1, 30}},
		{player: linked(address, "second", "tr"), rng: rng, burst: [2]int{60, 120}, pause: [2]int{1, 30}},
	}

	var seen crowd
	for second := range 3 * 3600 {
		var clicking []player
		for _, p := range pair {
			if p.clicks(second) {
				clicking = append(clicking, p.player)
			}
		}
		h.second(&seen, clicking)
	}

	assert.Equal(t, detect.Clear, seen.strongest, "their seconds meet as often as chance says")
}

func TestSlowTurnsAreCertainOnlyOnceChanceIsSure(t *testing.T) {
	h := newHarness(production())

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")
	slow := func(slots int) {
		for range slots {
			h.slot(turn{one, 0, 30}, turn{two, 57, 60})
		}
	}

	slow(10)
	assert.Equal(t, detect.Suspect, h.read(one).verdict, "30 and 3 seconds: ten spans are 15 seconds by chance")

	slow(9)
	assert.Equal(t, detect.Suspect, h.read(one).verdict)

	slow(1)
	assert.Equal(t, detect.Certain, h.read(one).verdict)
}

func TestACertainReadingNamesOnlyTheAccountsItIsSureOf(t *testing.T) {
	h := newHarness(production())

	one, two, three := linked(line, "one", "hu"), linked(line, "two", "hu"), linked(line, "three", "hu")

	for i := range 20 {
		turns := []turn{{one, 20, 80}, {two, 82, 142}}
		if i < 10 {
			turns = append(turns, turn{three, 10, 14})
		}
		h.slot(turns...)
	}

	r := h.read(one)
	require.Equal(t, detect.Certain, r.verdict)
	assert.Equal(t, []detect.Accomplice{{Account: "two", SignedIn: true, Scope: line}}, r.evidence.Accomplices)
	assert.Contains(t, r.evidence.Fields, detect.Field{Key: "accounts", Value: 2})
	assert.Equal(t, detect.Suspect, h.read(three).verdict, "ten spans of four seconds beside forty of one")
}

func TestUnsetBoundsNeverRead(t *testing.T) {
	unset := production()
	unset.MinSpans, unset.CertainSpans = 0, 0
	h := newHarness(unset)

	one, two := linked(line, "one", "hu"), linked(line, "two", "hu")

	h.turns(200, one, two)

	assert.Equal(t, detect.Clear, h.read(one).verdict)
}

func TestTheTurnsBeforeAWarningAreForgotten(t *testing.T) {
	h := newHarness(production())
	main := linked(address, "main", "fr")
	alt := guest(line, "alt", "fr")
	h.turns(30, main, alt)
	require.Equal(t, detect.Certain, h.read(main).verdict)

	h.watchdog.ForgetTurns("main", h.clock.Now())

	assert.Equal(t, detect.Clear, h.read(main).verdict, "its own reading is judged again at once")
	assert.Equal(t, detect.Clear, h.read(alt).verdict, "a partner's reading is judged again at once")

	h.turns(9, main, alt)
	assert.Equal(t, detect.Clear, h.read(alt).verdict, "the turns since the warning count from zero")

	h.turns(1, main, alt)
	r := h.read(alt)
	assert.Equal(t, detect.Certain, r.verdict)
	assert.Contains(t, r.evidence.Fields, detect.Field{Key: "sharedSpans", Value: 10})
}

func TestForgettingAnAccountNeverSeenChangesNothing(t *testing.T) {
	h := newHarness(production())
	one, two := linked(address, "one", "fr"), linked(address, "two", "fr")
	h.turns(30, one, two)

	h.watchdog.ForgetTurns("nobody", h.clock.Now())

	assert.Equal(t, detect.Certain, h.read(one).verdict)
}
