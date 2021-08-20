package wineregdiff

import (
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRegCommands(t *testing.T) {
	reg1 := newTestReg(t, "testdata/example1.reg")
	reg2 := newTestReg(t, "testdata/example2.reg")

	diff, err := Diff(reg1, reg2)
	assert.NoError(t, err)

	for _, changesFor := range []ChangesFor{ChangesFor1, ChangesFor2} {
		log.Printf("--- %s", changesFor)
		cmds := GenerateRegCommands(diff, RegistryRootLocalMachine, changesFor, true)
		for _, cmd := range cmds {
			log.Printf("wine " + strings.Join(cmd.Args, " "))
		}
	}

}
