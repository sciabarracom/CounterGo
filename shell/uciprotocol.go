package shell

import (
	"bufio"
	"counter/engine"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type SearchService interface {
	Search(searchParams engine.SearchParams) engine.SearchInfo
}

type SearchServiceFactory func(name string) SearchService

type commandHandler func(uci *UciProtocol, args []string)

type UciProtocol struct {
	commands             map[string]commandHandler
	searchServiceFactory SearchServiceFactory
	searchService        SearchService
	positions            []*engine.Position
}

const (
	Name    = "Counter"
	Version = "1.99"
	Author  = "Vadim Chizhov"
)

func UciCommand(uci *UciProtocol, args []string) {
	fmt.Printf("id name %s %s\n", Name, Version)
	fmt.Printf("id author %s\n", Author)
	fmt.Println("uciok")
}

func SetOptionCommand(uci *UciProtocol, args []string) {

}

func IsReadyCommand(uci *UciProtocol, args []string) {
	fmt.Println("readyok")
}

func PositionCommand(uci *UciProtocol, args []string) {
	var token = args[0]
	var fen string
	var movesIndex = findIndexString(args, "moves")
	if token == "startpos" {
		fen = engine.InitialPositionFen
	} else if token == "fen" {
		if movesIndex == -1 {
			fen = strings.Join(args[1:], " ")
		} else {
			fen = strings.Join(args[1:movesIndex], " ")
		}
	} else {
		DebugUci("Wrong position command")
		return
	}
	var p = engine.NewPositionFromFEN(fen)
	if p == nil {
		DebugUci("Wrong fen")
		return
	}
	var positions = []*engine.Position{p}
	if movesIndex >= 0 && movesIndex+1 < len(args) {
		for _, smove := range args[movesIndex+1:] {
			fmt.Println(smove)
			var move = engine.ParseMove(smove)
			var newPos = positions[len(positions)-1].MakeMoveIfLegal(move)
			if newPos == nil {
				DebugUci("Wrong move")
				return
			} else {
				positions = append(positions, newPos)
			}
		}
	}
	uci.positions = positions
}

func findIndexString(slice []string, value string) int {
	for p, v := range slice {
		if v == value {
			return p
		}
	}
	return -1
}

func GoCommand(uci *UciProtocol, args []string) {
	var limits = ParseLimits(args)
	var searchParams = engine.SearchParams{
		Positions: uci.positions,
		Limits:    limits,
		Progress:  engine.SendProgressToUci,
	}
	go func() {
		var searchResult = uci.searchService.Search(searchParams)
		engine.SendResultToUci(searchResult)
	}()
}

func ParseLimits(args []string) (result engine.LimitsType) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "ponder":
			result.Ponder = true
		case "wtime":
			result.WhiteTime, _ = strconv.Atoi(args[i+1])
			i++
		case "btime":
			result.BlackTime, _ = strconv.Atoi(args[i+1])
			i++
		case "winc":
			result.WhiteIncrement, _ = strconv.Atoi(args[i+1])
			i++
		case "binc":
			result.BlackIncrement, _ = strconv.Atoi(args[i+1])
			i++
		case "movestogo":
			result.MovesToGo, _ = strconv.Atoi(args[i+1])
			i++
		case "depth":
			result.Depth, _ = strconv.Atoi(args[i+1])
			i++
		case "nodes":
			result.Nodes, _ = strconv.Atoi(args[i+1])
			i++
		case "mate":
			result.Mate, _ = strconv.Atoi(args[i+1])
			i++
		case "movetime":
			result.MoveTime, _ = strconv.Atoi(args[i+1])
			i++
		case "infinite":
			result.Infinite = true
		}
	}
	return
}

func UciNewGameCommand(uci *UciProtocol, args []string) {
	uci.searchService = uci.searchServiceFactory("")
}

func PonderhitCommand(uci *UciProtocol, args []string) {

}

func StopCommand(uci *UciProtocol, args []string) {

}

func BenchmarkCommand(uci *UciProtocol, args []string) {
	var fen = "r1bqk2r/pppp1ppp/2n2n2/8/1b1NP3/2N5/PPP2PPP/R1BQKB1R w KQkq - 3 6 "
	var p = engine.NewPositionFromFEN(fen)
	var ml = &engine.MoveList{}
	const count = 100000000
	var start = time.Now()
	for i := 0; i < count; {
		ml.GenerateMoves(p)
		i += ml.Count
	}
	var elapsed = time.Since(start)
	fmt.Println(ml)
	fmt.Println(elapsed)
}

func PerftCommand(uci *UciProtocol, args []string) {
	var position = engine.NewPositionFromFEN(engine.InitialPositionFen)
	var depth = 6

	var start = time.Now()
	var n = Perft(position, depth)
	var elapsed = time.Since(start)

	var expected = []int{20, 400, 8902, 197281, 4865609, 119060324}
	var ok = n == expected[depth-1]
	fmt.Printf("Nodes: %v OK: %v Duration: %v\n", n, ok, elapsed)
}

func Perft(p *engine.Position, depth int) int {
	result := 0
	var ml engine.MoveList
	var child engine.Position
	ml.GenerateMoves(p)
	for i := 0; i < ml.Count; i++ {
		var move = ml.Items[i].Move
		if p.MakeMove(move, &child) {
			if depth > 1 {
				result += Perft(&child, depth-1)
			} else {
				result++
			}
		}
	}
	return result
}

func EvalCommand(uci *UciProtocol, args []string) {
	engine.TraceEvalSettings()
	var score = engine.Evaluate(uci.positions[len(uci.positions)-1])
	fmt.Printf("score %v\n", score)
}

func MoveCommand(uci *UciProtocol, args []string) {
	var move = engine.ParseMove(args[0])
	var newPos = uci.positions[len(uci.positions)-1].MakeMoveIfLegal(move)
	if newPos == nil {
		DebugUci("Wrong move")
		return
	}
	uci.positions = append(uci.positions, newPos)
	var limits = engine.LimitsType{
		MoveTime: 3000,
	}
	var searchParams = engine.SearchParams{
		Positions:      uci.positions,
		Limits:         limits,
		IsTraceEnabled: true,
		Progress:       engine.SendProgressToUci,
	}
	var searchResult = uci.searchService.Search(searchParams)
	engine.SendResultToUci(searchResult)
	newPos = newPos.MakeMoveIfLegal(searchResult.MainLine.Move)
	if newPos != nil {
		uci.positions = append(uci.positions, newPos)
		PrintPosition(newPos)
		fmt.Println(newPos)
	}
}

func EpdCommand(uci *UciProtocol, args []string) {
	var filePath = "tests.epd"
	var searchService = uci.searchServiceFactory("")
	RunEpdTest(filePath, searchService)
}

func ArenaCommand(uci *UciProtocol, args []string) {
	RunTournament(uci.searchServiceFactory)
}

func StatusCommand(uci *UciProtocol, args []string) {

}

func NewUciProtocol(searchServiceFactory SearchServiceFactory) *UciProtocol {
	var uci = &UciProtocol{}
	uci.commands = map[string]commandHandler{
		// UCI commands
		"uci":        UciCommand,
		"setoption":  SetOptionCommand,
		"isready":    IsReadyCommand,
		"position":   PositionCommand,
		"go":         GoCommand,
		"ucinewgame": UciNewGameCommand,
		"ponderhit":  PonderhitCommand,
		"stop":       StopCommand,

		// My commands
		"benchmark": BenchmarkCommand,
		"perft":     PerftCommand,
		"eval":      EvalCommand,
		"move":      MoveCommand,
		"epd":       EpdCommand,
		"arena":     ArenaCommand,
		"status":    StatusCommand,
	}
	uci.searchServiceFactory = searchServiceFactory
	uci.searchService = searchServiceFactory("")
	var p = engine.NewPositionFromFEN(engine.InitialPositionFen)
	uci.positions = []*engine.Position{p}
	return uci
}

func (uci *UciProtocol) Run() {
	fmt.Printf("%v %v\n", Name, Version)
	var scanner = bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var commandLine = scanner.Text()
		if commandLine == "quit" {
			return
		}
		var cmdArgs = strings.Split(commandLine, " ")
		var commandName = cmdArgs[0]
		var cmd, ok = uci.commands[commandName]
		if ok {
			cmd(uci, cmdArgs[1:])
		} else {
			DebugUci("Command not found.")
		}
	}
}

func DebugUci(s string) {
	fmt.Println("info string " + s)
}
