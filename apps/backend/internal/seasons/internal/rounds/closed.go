package rounds

type Closed struct {
	Round   Round
	Number  uint32
	Results []Result
}

func (r Round) Closed(number uint32, snapshot Snapshot) Closed {
	return Closed{Round: r, Number: number, Results: r.Results(snapshot)}
}
