//go:build testing

package reactions

import "github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"

func CountOf(reaction Reaction, total int, mine bool, reactors []Reactor) Count {
	return Count{reaction: reaction, total: total, mine: mine, reactors: reactors}
}

func TallyFor(id messages.MessageID, counts []Count, version uint64) Tally {
	return Tally{messageID: id, counts: counts, version: version}
}
