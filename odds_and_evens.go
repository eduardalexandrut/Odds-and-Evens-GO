package main

import (
	"fmt"
	"math/rand/v2"
)

// PlayerInfo Each player has its own channels:
// - play:   the referee sends a channel on which the player must answer with its throw
// - result: the referee tells the player if it won (true) or lost (false)
type PlayerInfo struct {
	id     int
	play   chan chan int
	result chan bool
}

type MatchResult struct {
	matchIdx int
	winner   PlayerInfo
}

// Player is a goroutine that lives for the whole tournament.
// It keeps playing until it loses, or until the coordinator closes its play
// channel (that means it is the champion).
func Player(p PlayerInfo, exited chan<- int) {
	for reply := range p.play {
		move := rand.IntN(6) // 0..5 fingers
		fmt.Printf("[Player %d] throws %d\n", p.id, move)
		reply <- move

		won := <-p.result
		if !won {
			fmt.Printf("[Player %d] eliminated\n", p.id)
			exited <- p.id
			return
		}
	}

	fmt.Printf("[Player %d] I am the champion!\n", p.id)
	exited <- p.id
}

// playMatch is the referee of a single match.
// p1 plays "odds", p2 plays "evens".
func playMatch(matchIdx int, p1, p2 PlayerInfo, roundNum int, resultChan chan MatchResult) {
	matchName := fmt.Sprintf("Round %d, Match %d", roundNum, matchIdx+1)

	// private channels: each player answers only to the referee
	moveChan1 := make(chan int)
	moveChan2 := make(chan int)

	p1.play <- moveChan1
	p2.play <- moveChan2

	m1 := <-moveChan1
	m2 := <-moveChan2
	sum := m1 + m2

	var winner PlayerInfo
	var loser PlayerInfo

	if sum%2 == 1 {
		winner, loser = p1, p2 // odd sum: the odds player wins
	} else {
		winner, loser = p2, p1 // even sum: the evens player wins
	}

	fmt.Printf("[%s] Player %d (odds) %d + Player %d (evens) %d = %d -> Player %d wins\n",
		matchName, p1.id, m1, p2.id, m2, sum, winner.id)

	// tell both players how it went
	loser.result <- false
	winner.result <- true

	// send completion message to the coordinator
	resultChan <- MatchResult{matchIdx: matchIdx, winner: winner}
}

func Coord(nPlayers int) {
	// buffered, so eliminated players can always exit without waiting
	exited := make(chan int, nPlayers)

	players := make([]PlayerInfo, nPlayers)
	for i := 0; i < nPlayers; i++ {
		players[i] = PlayerInfo{
			id:     i + 1,
			play:   make(chan chan int),
			result: make(chan bool),
		}
		go Player(players[i], exited)
	}

	roundNum := 1

	for len(players) > 1 {
		fmt.Printf("\n--- Start Round %d (Players left %d) ---\n", roundNum, len(players))

		nMatches := len(players) / 2
		nextRoundPlayers := make([]PlayerInfo, nMatches)

		// Buffered channel to receive match messages
		resultChan := make(chan MatchResult, nMatches)

		// launch matches in parallel
		for i := 0; i < len(players); i += 2 {
			go playMatch(i/2, players[i], players[i+1], roundNum, resultChan)
		}

		// wait messages of end match reading from channel nMatches times
		for j := 0; j < nMatches; j++ {
			res := <-resultChan
			nextRoundPlayers[res.matchIdx] = res.winner
		}

		players = nextRoundPlayers
		roundNum++
	}

	// tell the champion that the tournament is over
	close(players[0].play)

	// wait until every player goroutine has finished
	for i := 0; i < nPlayers; i++ {
		<-exited
	}

	fmt.Printf("\n !!! THE WINNER OF THE CHAMPIONSHIP IS %d!\n", players[0].id)
}

func main() {
	var nPlayers int

	for {
		fmt.Print("Enter the number of players (must be a power of 2): ")
		_, err := fmt.Scanln(&nPlayers)
		if err != nil {
			// throw away the rest of the wrong line, otherwise we loop forever
			var discard string
			fmt.Scanln(&discard)
			fmt.Println("Invalid input. Please enter a positive integer.")
			continue
		}

		if nPlayers < 2 || (nPlayers&(nPlayers-1)) != 0 {
			fmt.Println("The number of players must be a power of 2 (e.g., 2, 4, 8, 16). Try again.")
			continue
		}

		break
	}

	fmt.Printf("Start Odds-and-Evens: %d players\n", nPlayers)

	Coord(nPlayers)
}
