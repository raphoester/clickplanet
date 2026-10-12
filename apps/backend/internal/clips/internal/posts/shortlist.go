package posts

// From 1: what the generator's --pick takes.
type Pick int

type Story struct {
	Pick  Pick
	Title string
}

type Shortlist []Story

func ShortlistOf(titles []string) Shortlist {
	stories := make(Shortlist, 0, len(titles))
	for i, title := range titles {
		stories = append(stories, Story{Pick: Pick(i + 1), Title: title})
	}
	return stories
}

func (s Shortlist) Best(count int) Shortlist {
	return s[:min(max(count, 0), len(s))]
}

func (s Shortlist) Picks() []Pick {
	picks := make([]Pick, 0, len(s))
	for _, story := range s {
		picks = append(picks, story.Pick)
	}
	return picks
}
