// Package quizdata holds the quiz bank, copied from the monorepo-shared /quiz by `make quiz` —
// this app's half of that source, the way generated/map is its half of /map.
//
// It is this app's half and **nobody else's**: the frontend has no copy on purpose, because the
// answers are in here. A bank served to the page is a bank anyone can fetch with the network tab
// open. See /quiz/README.md.
//
// Embedded rather than read from disk, for the reason generated/map is: cmd/api is a self-contained
// container, and a file it has to find at boot is a boot that can fail for a reason nothing in this
// repo controls.
package quizdata

import _ "embed"

//go:embed bank.json
var bank []byte

func Bank() []byte { return bank }
