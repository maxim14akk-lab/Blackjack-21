// blackjack.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	reset  = "\033[0m"
	red    = "\033[91m"
	green  = "\033[92m"
	yellow = "\033[93m"
	blue   = "\033[94m"
	cyan   = "\033[96m"
	bold   = "\033[1m"
)

func colorize(text, color string) string {
	return color + text + reset
}

var suits = []string{"♠", "♥", "♦", "♣"}
var ranks = []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
var values = map[string]int{"2":2, "3":3, "4":4, "5":5, "6":6, "7":7, "8":8, "9":9, "10":10, "J":10, "Q":10, "K":10, "A":11}

type Card struct {
	Suit  string
	Rank  string
	Value int
}

func (c Card) String() string {
	col := red
	if c.Suit != "♥" && c.Suit != "♦" {
		col = reset
	}
	return colorize(c.Rank+c.Suit, col)
}

type Deck struct {
	Cards []Card
}

func NewDeck() *Deck {
	d := &Deck{}
	for _, s := range suits {
		for _, r := range ranks {
			d.Cards = append(d.Cards, Card{Suit: s, Rank: r, Value: values[r]})
		}
	}
	d.Shuffle()
	return d
}

func (d *Deck) Shuffle() {
	rand.Seed(time.Now().UnixNano())
	for i := range d.Cards {
		j := rand.Intn(i + 1)
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	}
}

func (d *Deck) Draw() Card {
	if len(d.Cards) == 0 {
		return Card{}
	}
	c := d.Cards[len(d.Cards)-1]
	d.Cards = d.Cards[:len(d.Cards)-1]
	return c
}

type Hand struct {
	Cards []Card
	Bet   int
	Done  bool
}

func (h *Hand) Value() int {
	total := 0
	aces := 0
	for _, c := range h.Cards {
		total += c.Value
		if c.Rank == "A" {
			aces++
		}
	}
	for total > 21 && aces > 0 {
		total -= 10
		aces--
	}
	return total
}

func (h *Hand) IsBlackjack() bool {
	return len(h.Cards) == 2 && h.Value() == 21
}

func (h *Hand) IsBust() bool {
	return h.Value() > 21
}

func (h *Hand) CanSplit() bool {
	return len(h.Cards) == 2 && h.Cards[0].Rank == h.Cards[1].Rank
}

func (h *Hand) Split() Hand {
	nh := Hand{Cards: []Card{h.Cards[1]}}
	h.Cards = h.Cards[:1]
	return nh
}

func (h *Hand) String() string {
	var s []string
	for _, c := range h.Cards {
		s = append(s, c.String())
	}
	return strings.Join(s, " ")
}

type Stats struct {
	Wins      int `json:"wins"`
	Losses    int `json:"losses"`
	Pushes    int `json:"pushes"`
	Blackjacks int `json:"blackjacks"`
}

type Blackjack struct {
	Balance      int
	Bet          int
	Deck         *Deck
	PlayerHands  []Hand
	DealerHand   Hand
	Insurance    bool
	InsurancePaid int
	Stats        Stats
	StatsFile    string
}

func NewBlackjack(balance int) *Blackjack {
	b := &Blackjack{
		Balance: balance,
		StatsFile: filepath.Join(os.Getenv("HOME"), ".blackjack_stats.json"),
	}
	b.loadStats()
	return b
}

func (b *Blackjack) loadStats() {
	data, err := os.ReadFile(b.StatsFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &b.Stats)
}

func (b *Blackjack) saveStats() {
	data, _ := json.MarshalIndent(b.Stats, "", "  ")
	os.WriteFile(b.StatsFile, data, 0644)
}

func (b *Blackjack) displayStats() {
	fmt.Println(colorize("Статистика:", bold))
	fmt.Printf("  Побед: %d\n", b.Stats.Wins)
	fmt.Printf("  Поражений: %d\n", b.Stats.Losses)
	fmt.Printf("  Ничьих: %d\n", b.Stats.Pushes)
	fmt.Printf("  Блэкджеков: %d\n", b.Stats.Blackjacks)
	fmt.Printf("  Баланс: %d\n", b.Balance)
}

func (b *Blackjack) dealInitial() {
	b.Deck = NewDeck()
	b.Deck.Shuffle()
	b.PlayerHands = []Hand{{}}
	b.DealerHand = Hand{}
	b.Insurance = false
	b.InsurancePaid = 0
	b.PlayerHands[0].Cards = append(b.PlayerHands[0].Cards, b.Deck.Draw())
	b.DealerHand.Cards = append(b.DealerHand.Cards, b.Deck.Draw())
	b.PlayerHands[0].Cards = append(b.PlayerHands[0].Cards, b.Deck.Draw())
	b.DealerHand.Cards = append(b.DealerHand.Cards, b.Deck.Draw())
}

func (b *Blackjack) showHands(hideDealer bool) {
	fmt.Println("\nДилер:")
	if hideDealer {
		fmt.Printf("  %s  [скрыто]\n", b.DealerHand.Cards[0].String())
	} else {
		fmt.Printf("  %s (очки: %d)\n", b.DealerHand.String(), b.DealerHand.Value())
	}
	fmt.Println("\nВаши руки:")
	for i, h := range b.PlayerHands {
		fmt.Printf("  Рука %d: %s (очки: %d)\n", i+1, h.String(), h.Value())
	}
}

func (b *Blackjack) getBet() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Ваша ставка (баланс: %d): ", b.Balance)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		bet, err := strconv.Atoi(input)
		if err == nil && bet > 0 && bet <= b.Balance {
			b.Bet = bet
			return
		}
		fmt.Println(colorize("Неверная ставка.", red))
	}
}

func (b *Blackjack) playHand(idx int) {
	reader := bufio.NewReader(os.Stdin)
	hand := &b.PlayerHands[idx]
	for !hand.Done && !hand.IsBust() && hand.Value() < 21 {
		b.showHands()
		fmt.Printf("\nРука %d: %s (очки: %d)\n", idx+1, hand.String(), hand.Value())
		fmt.Print("Действие (h=hit, s=stand, d=double, i=insurance, sp=split, q=quit): ")
		action, _ := reader.ReadString('\n')
		action = strings.TrimSpace(strings.ToLower(action))
		switch action {
		case "q":
			fmt.Println(colorize("Выход.", yellow))
			b.saveStats()
			os.Exit(0)
		case "h":
			hand.Cards = append(hand.Cards, b.Deck.Draw())
			if hand.IsBust() {
				fmt.Println(colorize("Перебор!", red))
				hand.Done = true
			}
		case "s":
			hand.Done = true
		case "d":
			if len(hand.Cards) == 2 && b.Balance >= b.Bet {
				b.Bet *= 2
				hand.Cards = append(hand.Cards, b.Deck.Draw())
				hand.Done = true
				if hand.IsBust() {
					fmt.Println(colorize("Перебор!", red))
				}
			} else {
				fmt.Println(colorize("Удвоение недоступно.", yellow))
			}
		case "i":
			if b.DealerHand.Cards[0].Rank == "A" && !b.Insurance {
				b.Insurance = true
				b.InsurancePaid = b.Bet / 2
				b.Balance -= b.InsurancePaid
				fmt.Println(colorize("Страховка активирована.", cyan))
			} else {
				fmt.Println(colorize("Страховка недоступна.", yellow))
			}
		case "sp":
			if hand.CanSplit() {
				newHand := hand.Split()
				b.PlayerHands = append(b.PlayerHands, newHand)
				hand.Cards = append(hand.Cards, b.Deck.Draw())
				newHand.Cards = append(newHand.Cards, b.Deck.Draw())
				fmt.Println(colorize("Руки разделены.", cyan))
				if b.Balance < b.Bet {
					fmt.Println(colorize("Недостаточно средств для сплита.", red))
					b.PlayerHands = b.PlayerHands[:len(b.PlayerHands)-1]
				} else {
					b.Balance -= b.Bet
					b.Bet *= 2
				}
			} else {
				fmt.Println(colorize("Сплит недоступен.", yellow))
			}
		default:
			fmt.Println(colorize("Неизвестная команда.", red))
		}
	}
}

func (b *Blackjack) dealerPlay() {
	for b.DealerHand.Value() < 17 {
		b.DealerHand.Cards = append(b.DealerHand.Cards, b.Deck.Draw())
	}
}

func (b *Blackjack) resolve() {
	b.showHands(false)
	dealerVal := b.DealerHand.Value()
	for _, hand := range b.PlayerHands {
		playerVal := hand.Value()
		if hand.IsBust() {
			b.Balance -= b.Bet
			b.Stats.Losses++
			fmt.Printf(colorize("Поражение (перебор): рука проиграла %d\n", red), b.Bet)
		} else if hand.IsBlackjack() && !b.DealerHand.IsBlackjack() {
			win := int(float64(b.Bet) * 1.5)
			b.Balance += win
			b.Stats.Blackjacks++
			b.Stats.Wins++
			fmt.Printf(colorize("Блэкджек! Выигрыш %d\n", green), win)
		} else if dealerVal > 21 {
			b.Balance += b.Bet
			b.Stats.Wins++
			fmt.Printf(colorize("Дилер перебрал, выигрыш %d\n", green), b.Bet)
		} else if playerVal > dealerVal {
			b.Balance += b.Bet
			b.Stats.Wins++
			fmt.Printf(colorize("Выигрыш %d\n", green), b.Bet)
		} else if playerVal == dealerVal {
			b.Stats.Pushes++
			fmt.Println(colorize("Ничья, ставка возвращена.", yellow))
		} else {
			b.Balance -= b.Bet
			b.Stats.Losses++
			fmt.Printf(colorize("Поражение, потеряно %d\n", red), b.Bet)
		}
	}
	if b.Insurance {
		if b.DealerHand.IsBlackjack() {
			b.Balance += b.InsurancePaid * 2
			fmt.Println(colorize("Страховка выиграла!", cyan))
		} else {
			fmt.Println(colorize("Страховка проиграла.", yellow))
		}
	}
	b.saveStats()
}

func (b *Blackjack) play() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(colorize("🃏 Добро пожаловать в Блэкджек!", bold))
	fmt.Printf("Ваш баланс: %d\n", b.Balance)
	for b.Balance > 0 {
		b.getBet()
		b.dealInitial()
		// Блэкджек игрока
		if b.PlayerHands[0].IsBlackjack() {
			b.showHands()
			if b.DealerHand.IsBlackjack() {
				fmt.Println(colorize("Ничья – у обоих блэкджек!", yellow))
				b.Stats.Pushes++
				b.saveStats()
				continue
			} else {
				win := int(float64(b.Bet) * 1.5)
				b.Balance += win
				b.Stats.Blackjacks++
				b.Stats.Wins++
				fmt.Printf(colorize("Блэкджек! Выигрыш %d\n", green), win)
				b.saveStats()
				continue
			}
		}
		// Страховка
		if b.DealerHand.Cards[0].Rank == "A" {
			fmt.Print("У дилера туз. Хотите страховку? (y/n): ")
			ans, _ := reader.ReadString('\n')
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans == "y" {
				b.Insurance = true
				b.InsurancePaid = b.Bet / 2
				b.Balance -= b.InsurancePaid
				fmt.Println(colorize("Страховка активирована.", cyan))
			}
		}
		// Ход игрока
		for i := range b.PlayerHands {
			b.playHand(i)
		}
		// Ход дилера
		b.dealerPlay()
		b.resolve()
		fmt.Printf(colorize("Баланс: %d\n", bold), b.Balance)
		if b.Balance <= 0 {
			fmt.Println(colorize("Вы проиграли все деньги!", red))
			break
		}
		fmt.Print("Продолжить игру? (y/n): ")
		cont, _ := reader.ReadString('\n')
		cont = strings.TrimSpace(strings.ToLower(cont))
		if cont != "y" {
			break
		}
	}
	b.displayStats()
	b.saveStats()
}

func main() {
	balance := 1000
	reset := false
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-r" || arg == "--reset" {
			reset = true
		} else if arg == "-s" || arg == "--start" {
			if i+1 < len(os.Args) {
				balance, _ = strconv.Atoi(os.Args[i+1])
				i++
			}
		} else if arg == "-h" || arg == "--help" {
			fmt.Println("Usage: blackjack [-s start_balance] [-r]")
			return
		}
	}
	if reset {
		f := filepath.Join(os.Getenv("HOME"), ".blackjack_stats.json")
		os.Remove(f)
		fmt.Println("Статистика сброшена.")
		return
	}
	game := NewBlackjack(balance)
	game.play()
}
