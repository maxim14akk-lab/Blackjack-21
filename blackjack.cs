// blackjack.cs
using System;
using System.Collections.Generic;
using System.IO;
using System.Text.Json;
using System.Linq;

class BlackjackGame
{
    static string Colorize(string text, string color)
    {
        string col = color switch
        {
            "red" => "\x1b[91m",
            "green" => "\x1b[92m",
            "yellow" => "\x1b[93m",
            "blue" => "\x1b[94m",
            "cyan" => "\x1b[96m",
            "bold" => "\x1b[1m",
            _ => "\x1b[0m"
        };
        return col + text + "\x1b[0m";
    }

    static string[] SUITS = {"♠","♥","♦","♣"};
    static string[] RANKS = {"2","3","4","5","6","7","8","9","10","J","Q","K","A"};
    static Dictionary<string, int> VALUES = new Dictionary<string, int> {
        {"2",2},{"3",3},{"4",4},{"5",5},{"6",6},{"7",7},{"8",8},{"9",9},{"10",10},
        {"J",10},{"Q",10},{"K",10},{"A",11}
    };

    class Card
    {
        public string Suit { get; }
        public string Rank { get; }
        public int Value { get; }
        public Card(string suit, string rank)
        {
            Suit = suit; Rank = rank; Value = VALUES[rank];
        }
        public override string ToString()
        {
            string col = (Suit == "♥" || Suit == "♦") ? "red" : "reset";
            return Colorize(Rank + Suit, col);
        }
    }

    class Deck
    {
        public List<Card> Cards { get; set; }
        public Deck()
        {
            Cards = new List<Card>();
            foreach (var s in SUITS)
                foreach (var r in RANKS)
                    Cards.Add(new Card(s, r));
            Shuffle();
        }
        public void Shuffle()
        {
            Random rnd = new Random();
            for (int i = Cards.Count - 1; i > 0; i--)
            {
                int j = rnd.Next(i + 1);
                var tmp = Cards[i];
                Cards[i] = Cards[j];
                Cards[j] = tmp;
            }
        }
        public Card Draw()
        {
            if (Cards.Count == 0) return null;
            var c = Cards[Cards.Count - 1];
            Cards.RemoveAt(Cards.Count - 1);
            return c;
        }
    }

    class Hand
    {
        public List<Card> Cards { get; set; } = new List<Card>();
        public int Bet { get; set; }
        public bool Done { get; set; }
        public int Value()
        {
            int total = Cards.Sum(c => c.Value);
            int aces = Cards.Count(c => c.Rank == "A");
            while (total > 21 && aces > 0) { total -= 10; aces--; }
            return total;
        }
        public bool IsBlackjack() => Cards.Count == 2 && Value() == 21;
        public bool IsBust() => Value() > 21;
        public bool CanSplit() => Cards.Count == 2 && Cards[0].Rank == Cards[1].Rank;
        public Hand Split()
        {
            Hand h = new Hand();
            h.Cards.Add(Cards[Cards.Count - 1]);
            Cards.RemoveAt(Cards.Count - 1);
            return h;
        }
        public override string ToString() => string.Join(" ", Cards.Select(c => c.ToString()));
    }

    class Stats
    {
        public int wins { get; set; }
        public int losses { get; set; }
        public int pushes { get; set; }
        public int blackjacks { get; set; }
    }

    class Blackjack
    {
        public int Balance { get; set; }
        public int Bet { get; set; }
        public Deck Deck { get; set; }
        public List<Hand> PlayerHands { get; set; }
        public Hand DealerHand { get; set; }
        public bool Insurance { get; set; }
        public int InsurancePaid { get; set; }
        public Stats Stats { get; set; }
        public string StatsFile { get; }

        public Blackjack(int balance)
        {
            Balance = balance;
            StatsFile = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), ".blackjack_stats.json");
            LoadStats();
        }

        void LoadStats()
        {
            if (File.Exists(StatsFile))
            {
                string json = File.ReadAllText(StatsFile);
                Stats = JsonSerializer.Deserialize<Stats>(json) ?? new Stats();
            }
            else Stats = new Stats();
        }

        void SaveStats()
        {
            string json = JsonSerializer.Serialize(Stats);
            File.WriteAllText(StatsFile, json);
        }

        void DisplayStats()
        {
            Console.WriteLine(Colorize("Статистика:", "bold"));
            Console.WriteLine($"  Побед: {Stats.wins}");
            Console.WriteLine($"  Поражений: {Stats.losses}");
            Console.WriteLine($"  Ничьих: {Stats.pushes}");
            Console.WriteLine($"  Блэкджеков: {Stats.blackjacks}");
            Console.WriteLine($"  Баланс: {Balance}");
        }

        void DealInitial()
        {
            Deck = new Deck();
            Deck.Shuffle();
            PlayerHands = new List<Hand> { new Hand() };
            DealerHand = new Hand();
            Insurance = false;
            InsurancePaid = 0;
            PlayerHands[0].Cards.Add(Deck.Draw());
            DealerHand.Cards.Add(Deck.Draw());
            PlayerHands[0].Cards.Add(Deck.Draw());
            DealerHand.Cards.Add(Deck.Draw());
        }

        void ShowHands(bool hideDealer = true)
        {
            Console.WriteLine("\nДилер:");
            if (hideDealer)
                Console.WriteLine($"  {DealerHand.Cards[0].ToString()}  [скрыто]");
            else
                Console.WriteLine($"  {DealerHand} (очки: {DealerHand.Value()})");
            Console.WriteLine("\nВаши руки:");
            for (int i=0; i<PlayerHands.Count; i++)
                Console.WriteLine($"  Рука {i+1}: {PlayerHands[i]} (очки: {PlayerHands[i].Value()})");
        }

        void GetBet()
        {
            while (true)
            {
                Console.Write($"Ваша ставка (баланс: {Balance}): ");
                string input = Console.ReadLine();
                if (int.TryParse(input, out int bet) && bet > 0 && bet <= Balance)
                {
                    Bet = bet;
                    return;
                }
                Console.WriteLine(Colorize("Неверная ставка.", "red"));
            }
        }

        void PlayHand(int idx)
        {
            var hand = PlayerHands[idx];
            while (!hand.Done && !hand.IsBust() && hand.Value() < 21)
            {
                ShowHands();
                Console.WriteLine($"\nРука {idx+1}: {hand} (очки: {hand.Value()})");
                Console.Write("Действие (h=hit, s=stand, d=double, i=insurance, sp=split, q=quit): ");
                string action = Console.ReadLine().Trim().ToLower();
                switch (action)
                {
                    case "q":
                        Console.WriteLine(Colorize("Выход.", "yellow"));
                        SaveStats();
                        Environment.Exit(0);
                        break;
                    case "h":
                        hand.Cards.Add(Deck.Draw());
                        if (hand.IsBust()) { Console.WriteLine(Colorize("Перебор!", "red")); hand.Done = true; }
                        break;
                    case "s":
                        hand.Done = true;
                        break;
                    case "d":
                        if (hand.Cards.Count == 2 && Balance >= Bet)
                        {
                            Bet *= 2;
                            hand.Cards.Add(Deck.Draw());
                            hand.Done = true;
                            if (hand.IsBust()) Console.WriteLine(Colorize("Перебор!", "red"));
                        }
                        else Console.WriteLine(Colorize("Удвоение недоступно.", "yellow"));
                        break;
                    case "i":
                        if (DealerHand.Cards[0].Rank == "A" && !Insurance)
                        {
                            Insurance = true;
                            InsurancePaid = Bet / 2;
                            Balance -= InsurancePaid;
                            Console.WriteLine(Colorize("Страховка активирована.", "cyan"));
                        }
                        else Console.WriteLine(Colorize("Страховка недоступна.", "yellow"));
                        break;
                    case "sp":
                        if (hand.CanSplit())
                        {
                            var newHand = hand.Split();
                            PlayerHands.Add(newHand);
                            hand.Cards.Add(Deck.Draw());
                            newHand.Cards.Add(Deck.Draw());
                            Console.WriteLine(Colorize("Руки разделены.", "cyan"));
                            if (Balance < Bet)
                            {
                                Console.WriteLine(Colorize("Недостаточно средств для сплита.", "red"));
                                PlayerHands.RemoveAt(PlayerHands.Count - 1);
                            }
                            else
                            {
                                Balance -= Bet;
                                Bet *= 2;
                            }
                        }
                        else Console.WriteLine(Colorize("Сплит недоступен.", "yellow"));
                        break;
                    default:
                        Console.WriteLine(Colorize("Неизвестная команда.", "red"));
                        break;
                }
            }
        }

        void DealerPlay()
        {
            while (DealerHand.Value() < 17)
                DealerHand.Cards.Add(Deck.Draw());
        }

        void Resolve()
        {
            ShowHands(false);
            int dealerVal = DealerHand.Value();
            foreach (var hand in PlayerHands)
            {
                int playerVal = hand.Value();
                if (hand.IsBust())
                {
                    Balance -= Bet;
                    Stats.losses++;
                    Console.WriteLine(Colorize($"Поражение (перебор): рука проиграла {Bet}", "red"));
                }
                else if (hand.IsBlackjack() && !DealerHand.IsBlackjack())
                {
                    int win = (int)(Bet * 1.5);
                    Balance += win;
                    Stats.blackjacks++;
                    Stats.wins++;
                    Console.WriteLine(Colorize($"Блэкджек! Выигрыш {win}", "green"));
                }
                else if (dealerVal > 21)
                {
                    Balance += Bet;
                    Stats.wins++;
                    Console.WriteLine(Colorize($"Дилер перебрал, выигрыш {Bet}", "green"));
                }
                else if (playerVal > dealerVal)
                {
                    Balance += Bet;
                    Stats.wins++;
                    Console.WriteLine(Colorize($"Выигрыш {Bet}", "green"));
                }
                else if (playerVal == dealerVal)
                {
                    Stats.pushes++;
                    Console.WriteLine(Colorize("Ничья, ставка возвращена.", "yellow"));
                }
                else
                {
                    Balance -= Bet;
                    Stats.losses++;
                    Console.WriteLine(Colorize($"Поражение, потеряно {Bet}", "red"));
                }
            }
            if (Insurance)
            {
                if (DealerHand.IsBlackjack())
                {
                    Balance += InsurancePaid * 2;
                    Console.WriteLine(Colorize("Страховка выиграла!", "cyan"));
                }
                else Console.WriteLine(Colorize("Страховка проиграла.", "yellow"));
            }
            SaveStats();
        }

        public void Play()
        {
            Console.WriteLine(Colorize("🃏 Добро пожаловать в Блэкджек!", "bold"));
            Console.WriteLine($"Ваш баланс: {Balance}");
            while (Balance > 0)
            {
                GetBet();
                DealInitial();
                if (PlayerHands[0].IsBlackjack())
                {
                    ShowHands();
                    if (DealerHand.IsBlackjack())
                    {
                        Console.WriteLine(Colorize("Ничья – у обоих блэкджек!", "yellow"));
                        Stats.pushes++;
                        SaveStats();
                        continue;
                    }
                    else
                    {
                        int win = (int)(Bet * 1.5);
                        Balance += win;
                        Stats.blackjacks++;
                        Stats.wins++;
                        Console.WriteLine(Colorize($"Блэкджек! Выигрыш {win}", "green"));
                        SaveStats();
                        continue;
                    }
                }
                if (DealerHand.Cards[0].Rank == "A")
                {
                    Console.Write("У дилера туз. Хотите страховку? (y/n): ");
                    string ans = Console.ReadLine().Trim().ToLower();
                    if (ans == "y")
                    {
                        Insurance = true;
                        InsurancePaid = Bet / 2;
                        Balance -= InsurancePaid;
                        Console.WriteLine(Colorize("Страховка активирована.", "cyan"));
                    }
                }
                for (int i=0; i<PlayerHands.Count; i++)
                    PlayHand(i);
                DealerPlay();
                Resolve();
                Console.WriteLine(Colorize($"Баланс: {Balance}", "bold"));
                if (Balance <= 0)
                {
                    Console.WriteLine(Colorize("Вы проиграли все деньги!", "red"));
                    break;
                }
                Console.Write("Продолжить игру? (y/n): ");
                string cont = Console.ReadLine().Trim().ToLower();
                if (cont != "y") break;
            }
            DisplayStats();
            SaveStats();
        }
    }

    static void Main(string[] args)
    {
        int balance = 1000;
        bool reset = false;
        for (int i=0; i<args.Length; i++)
        {
            if (args[i] == "-r" || args[i] == "--reset") reset = true;
            else if ((args[i] == "-s" || args[i] == "--start") && i+1 < args.Length)
                balance = int.Parse(args[++i]);
            else if (args[i] == "-h" || args[i] == "--help")
            {
                Console.WriteLine("Usage: blackjack [-s start_balance] [-r]");
                return;
            }
        }
        if (reset)
        {
            string f = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), ".blackjack_stats.json");
            if (File.Exists(f)) File.Delete(f);
            Console.WriteLine("Статистика сброшена.");
            return;
        }
        Blackjack game = new Blackjack(balance);
        game.Play();
    }
}
