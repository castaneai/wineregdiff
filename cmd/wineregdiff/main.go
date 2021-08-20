package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/castaneai/wineregdiff"
)

var (
	changesForIn = flag.String("changes-for", "1", `Controls the direction of the diff("1" or "2").`)
	rootIn       = flag.String("root", "HKLM", "Registry root key. HKLM|HKCU|HKCR|HKU|HKCC")
	force        = flag.Bool("force", false, "Add the /f flag to the generated command.")
)

func main() {
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		flag.Usage()
		os.Exit(2)
	}
	changesFor, err := parseChangesFor(*changesForIn)
	if err != nil {
		log.Println(err.Error())
		os.Exit(2)
	}
	regRoot, err := parseRoot(*rootIn)
	if err != nil {
		log.Println(err.Error())
		os.Exit(2)
	}
	if err := run(args[0], args[1], regRoot, changesFor, *force); err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
}

func usage() {
	log.Println("Usage of wineregdiff:")
	log.Println("\twineregdiff [Flags] [REGFILE1] [REGFILE2]")
	log.Println("Flags:")
	flag.PrintDefaults()
}

func run(file1, file2 string, root wineregdiff.RegistryRoot, changesFor wineregdiff.ChangesFor, force bool) error {
	reg1, err := parseRegFile(file1)
	if err != nil {
		return err
	}
	reg2, err := parseRegFile(file2)
	if err != nil {
		return err
	}
	diff, err := wineregdiff.Diff(reg1, reg2)
	if err != nil {
		return err
	}
	regCmds := wineregdiff.GenerateRegCommands(diff, root, changesFor, force)
	for _, cmd := range regCmds {
		log.Printf("wine %s", strings.Join(cmd.Args, " "))
	}
	return nil
}

func parseRegFile(filename string) (wineregdiff.Registry, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return wineregdiff.Parse(f)
}

func parseChangesFor(s string) (wineregdiff.ChangesFor, error) {
	switch s {
	case "1":
		return wineregdiff.ChangesFor1, nil
	case "2":
		return wineregdiff.ChangesFor2, nil
	default:
		return 0, fmt.Errorf("invalid changesFor ('1' or '2'): '%s'", s)
	}
}

func parseRoot(s string) (wineregdiff.RegistryRoot, error) {
	switch s {
	case "HKLM":
		return wineregdiff.RegistryRootLocalMachine, nil
	case "HKCU":
		return wineregdiff.RegistryRootCurrentUser, nil
	case "HKCR":
		return wineregdiff.RegistryRootClassesRoot, nil
	case "HKU":
		return wineregdiff.RegistryRootUsers, nil
	case "HKCC":
		return wineregdiff.RegistryRootCurrentConfig, nil
	default:
		return "", fmt.Errorf("invalid registry root: '%s'", s)
	}
}
