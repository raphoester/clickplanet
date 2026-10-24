package clips

import "github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"

// The experiments that run on the clips, and the only place that says so: `choose` puts every clip it picks in an arm
// of each, with a coin of its own, and the generator and `schedule` act on the arms. Stopping one is taking its entry
// out; starting one is an entry here, its arms in the generator's `src/clip/experiments.ts`, and the code that acts
// on them, at render time in the generator or at post time in the job (`posts.PostSnapshot.Variants`).
var running = []posts.Experiment{
	// Started 2026-10-24. as-chosen: as long as the camera has somewhere to go, 8.5 to 22s. short: the shortest a
	// story of its kind plays, 8.5s for a front, 10 for a day's podium; a tour is already its shortest.
	{Name: "length", Arms: posts.EvenArms("as-chosen", "short"), Control: "as-chosen"},
}
