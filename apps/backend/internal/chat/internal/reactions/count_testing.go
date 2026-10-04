//go:build testing

package reactions

import "github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"

func CountOf(reaction Reaction, total int, mine bool, reactors []Reactor, names []string) Count {
	return Count{reaction: reaction, total: total, mine: mine, reactors: reactors, names: names}
}

func TallyFor(id messages.MessageID, counts []Count, version uint64) Tally {
	return Tally{messageID: id, counts: counts, version: version}
}
