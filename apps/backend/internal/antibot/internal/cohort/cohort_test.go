package cohort_test

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/cohort"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
)

func newWatchdog() *cohort.Watchdog {
	return cohort.New(cohort.Config{
		StartWindow:    5 * time.Second,
		MinClicks:      20,
		MinFlagShare:   0.9,
		RateRatio:      1.5,
		LengthRatio:    1.2,
		QuietAfter:     time.Minute,
		MinMembers:     2,
		V4Bits:         24,
		V6Bits:         44,
		CertainCohorts: 3,
		CertainMembers: 6,
		ChainWindow:    30 * time.Minute,
	}, nil, nil)
}

func at(clock string) time.Time {
	t, err := time.Parse("15:04:05.000", clock)
	if err != nil {
		panic(err)
	}
	return time.Date(2026, 9, 14, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

type identity struct {
	scope  string
	first  time.Time
	stays  time.Duration
	gap    time.Duration
	jitter time.Duration
	flag   string
}

type click struct {
	scope string
	flag  string
	at    time.Time
}

func replay(seed uint64, identities ...identity) []click {
	//nolint:gosec // seeded test PRNG
	random := rand.New(rand.NewPCG(seed, seed+1))

	var clicks []click
	for _, id := range identities {
		end := id.first.Add(id.stays)
		for t := id.first; !t.After(end); {
			clicks = append(clicks, click{scope: id.scope, flag: id.flag, at: t})
			t = t.Add(id.gap + time.Duration(random.Int64N(int64(2*id.jitter)+1)) - id.jitter)
		}
	}

	sort.SliceStable(clicks, func(i, j int) bool { return clicks[i].at.Before(clicks[j].at) })
	return clicks
}

type reading struct {
	verdict      detect.Verdict
	firstCertain time.Time
	evidence     detect.Evidence
}

func run(w *cohort.Watchdog, clicks []click) map[string]reading {
	out := map[string]reading{}

	for _, c := range clicks {
		observed := detect.Click{Scope: c.scope, Tile: 1, Country: c.flag, At: c.at}
		w.Attempted(observed)
		verdict, evidence := w.Watch(observed)

		r := out[c.scope]
		if verdict > r.verdict {
			r.verdict, r.evidence = verdict, evidence
		}
		if verdict == detect.Certain && r.firstCertain.IsZero() {
			r.firstCertain = c.at
		}
		out[c.scope] = r
	}

	return out
}

func firefoxPool(bot func(scope, first string) identity) []identity {
	return []identity{
		bot("2a00:8c40:f0c2:11a0::/64", "19:43:18.204"),
		bot("2a00:8c40:f0c8:5e31::/64", "19:43:20.117"),
		bot("2a00:8c40:f0cd:0b7c::/64", "19:43:22.031"),

		bot("2a00:8c40:f0c7:a1a5::/64", "19:46:30.340"),
		bot("2a00:8c40:f0ce:fda7::/64", "19:46:30.980"),

		bot("2a00:8c40:f0c5:6713::/64", "19:54:32.516"),
		bot("2a00:8c40:f0c1:9190::/64", "19:54:32.521"),

		bot("2a00:8c40:f0c9:2d04::/64", "20:02:28.902"),
		bot("2a00:8c40:f0cb:7e12::/64", "20:02:28.907"),
	}
}

func poolBot(scope, first string) identity {
	return identity{
		scope:  scope,
		first:  at(first),
		stays:  476 * time.Second,
		gap:    2 * time.Second,
		jitter: 400 * time.Millisecond,
		flag:   "bg",
	}
}

func TestTheFirefoxVPNPoolIsCaught(t *testing.T) {
	readings := run(newWatchdog(), replay(1, firefoxPool(poolBot)...))

	for _, scope := range []string{
		"2a00:8c40:f0c2:11a0::/64", "2a00:8c40:f0c8:5e31::/64", "2a00:8c40:f0cd:0b7c::/64",
		"2a00:8c40:f0c7:a1a5::/64", "2a00:8c40:f0ce:fda7::/64",
	} {
		assert.Equal(t, detect.Suspect, readings[scope].verdict, scope)
	}

	for _, id := range []identity{
		poolBot("2a00:8c40:f0c5:6713::/64", "19:54:32.516"),
		poolBot("2a00:8c40:f0c1:9190::/64", "19:54:32.521"),
		poolBot("2a00:8c40:f0c9:2d04::/64", "20:02:28.902"),
		poolBot("2a00:8c40:f0cb:7e12::/64", "20:02:28.907"),
	} {
		r := readings[id.scope]
		require.Equal(t, detect.Certain, r.verdict, id.scope)
		assert.Equal(t, "chain", r.evidence.Rule)

		assert.Less(t, r.firstCertain.Sub(id.first), time.Minute, "certain as soon as it has clicked enough to compare")
	}
}

func TestAPoolThatStaggersItsStartsIsNotACohort(t *testing.T) {
	readings := run(newWatchdog(), replay(2,
		poolBot("2a00:8c40:f0c7:a1a5::/64", "19:46:30.000"),
		poolBot("2a00:8c40:f0ce:fda7::/64", "19:46:40.000"),
	))

	for scope, r := range readings {
		assert.Equal(t, detect.Clear, r.verdict, scope)
	}
}

func TestTwoPlayersJoiningAFlagWarTogetherAreOnlySuspect(t *testing.T) {
	player := func(scope string, gap time.Duration) identity {
		return identity{
			scope:  scope,
			first:  at("19:54:32.000"),
			stays:  30 * time.Minute,
			gap:    gap,
			jitter: 1500 * time.Millisecond,
			flag:   "bg",
		}
	}

	readings := run(newWatchdog(), replay(3,
		player("2a00:8c40:f0c5:6713::/64", 2*time.Second),
		player("2a00:8c40:f0c1:9190::/64", 2200*time.Millisecond),
	))

	for scope, r := range readings {
		assert.Equal(t, detect.Suspect, r.verdict, "%s: one group is never a chain", scope)
	}
}

func TestAPartnerWhoLeavesClearsTheOneWhoStays(t *testing.T) {
	w := newWatchdog()

	clicks := replay(4,
		identity{scope: "203.0.113.7", first: at("19:00:00.000"), stays: 3 * time.Minute, gap: 2 * time.Second, jitter: time.Second, flag: "bg"},
		identity{scope: "203.0.113.9", first: at("19:00:01.000"), stays: 20 * time.Minute, gap: 2 * time.Second, jitter: time.Second, flag: "bg"},
	)

	var last detect.Verdict
	for _, c := range clicks {
		observed := detect.Click{Scope: c.scope, Tile: 1, Country: c.flag, At: c.at}
		w.Attempted(observed)
		verdict, _ := w.Watch(observed)
		if c.scope == "203.0.113.9" {
			last = verdict
		}
	}

	assert.Equal(t, detect.Clear, last, "a pool's identities stop together; people do not")
}

func TestPlayersOnDifferentFlagsOrPacesAreClear(t *testing.T) {
	base := identity{first: at("19:00:00.000"), stays: 10 * time.Minute, gap: 2 * time.Second, jitter: 300 * time.Millisecond, flag: "bg"}

	with := func(scope string, change func(*identity)) identity {
		id := base
		id.scope = scope
		change(&id)
		return id
	}

	readings := run(newWatchdog(), replay(5,
		with("198.51.100.1", func(*identity) {}),
		with("198.51.100.2", func(id *identity) { id.flag = "ro" }),
		with("198.51.100.3", func(id *identity) { id.gap = 4 * time.Second }),
	))

	for scope, r := range readings {
		assert.Equal(t, detect.Clear, r.verdict, scope)
	}
}

func TestARaidFromAllOverIsNeverCertain(t *testing.T) {
	//nolint:gosec // seeded test PRNG
	random := rand.New(rand.NewPCG(6, 7))

	identities := make([]identity, 0, 6*12)
	for wave := range 6 {
		start := at("19:00:00.000").Add(time.Duration(wave) * 5 * time.Minute)
		for person := range 12 {
			identities = append(identities, identity{
				scope:  fmt.Sprintf("2001:db8:%x:%x::/64", random.IntN(0xffff), person),
				first:  start.Add(time.Duration(random.IntN(3000)) * time.Millisecond),
				stays:  8 * time.Minute,
				gap:    time.Second,
				jitter: 200 * time.Millisecond,
				flag:   "bg",
			})
		}
	}

	readings := run(newWatchdog(), replay(8, identities...))

	suspects := 0
	for scope, r := range readings {
		assert.NotEqual(t, detect.Certain, r.verdict, scope)
		if r.verdict == detect.Suspect {
			suspects++
		}
	}

	assert.Positive(t, suspects)
}

func TestManyScopesInStepFromOneRangeAreCertain(t *testing.T) {
	identities := make([]identity, 0, 6)
	for i := range 6 {
		identities = append(identities, poolBot(fmt.Sprintf("2a00:8c40:f0c%x:%x::/64", i, 0x100+i), "19:00:00.000"))
	}

	readings := run(newWatchdog(), replay(9, identities...))

	for scope, r := range readings {
		require.Equal(t, detect.Certain, r.verdict, scope)
		assert.Equal(t, "crowd", r.evidence.Rule)
	}
}

func TestScopesThatAreNotAddressesAreNeverCertain(t *testing.T) {
	identities := make([]identity, 0, 8)
	for i := range 8 {
		identities = append(identities, poolBot(fmt.Sprintf("caller-%d", i), "19:00:00.000"))
	}

	readings := run(newWatchdog(), replay(10, identities...))

	for scope, r := range readings {
		assert.Equal(t, detect.Suspect, r.verdict, scope)
	}
}

func TestValidateRefusesBoundsThatCannotMeanAnything(t *testing.T) {
	require.NoError(t, cohort.Config{}.Validate(), "every zero is a default")

	for name, config := range map[string]cohort.Config{
		"negative window":   {StartWindow: -time.Second},
		"share over one":    {MinFlagShare: 1.5},
		"ratio under one":   {RateRatio: 0.8},
		"cohort of one":     {MinMembers: 1},
		"chain of one":      {CertainCohorts: 1},
		"v4 prefix too big": {V4Bits: 33},
		"v6 below the /64":  {V6Bits: 96},
	} {
		assert.Error(t, config.Validate(), name)
	}
}
