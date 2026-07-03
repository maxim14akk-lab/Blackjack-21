// blackjack.cpp
#include <iostream>
#include <vector>
#include <string>
#include <random>
#include <algorithm>
#include <map>
#include <fstream>
#include <cctype>
#include <filesystem>

using namespace std;
namespace fs = std::filesystem;

const string RESET = "\033[0m";
const string RED = "\033[91m";
const string GREEN = "\033[92m";
const string YELLOW = "\033[93m";
const string BLUE = "\033[94m";
const string CYAN = "\033[96m";
const string BOLD = "\033[1m";

string colorize(const string& text, const string& color) {
    return color + text + RESET;
}

vector<string> SUITS = {"♠", "♥", "♦", "♣"};
vector<string> RANKS = {"2","3","4","5","6","7","8","9","10","J","Q","K","A"};
map<string, int> VALUES = {{"2",2},{"3",3},{"4",4},{"5",5},{"6",6},{"7",7},{"8",8},{"9",9},{"10",10},{"J",10},{"Q",10},{"K",10},{"A",11}};

struct Card {
    string suit;
    string rank;
    int value;
    string color;
    Card(string s, string r) : suit(s), rank(r), value(VALUES[r]) {
        color = (s == "♥" || s == "♦") ? RED : RESET;
    }
    string str() const {
        return colorize(rank + suit, color);
    }
};

class Deck {
public:
    vector<Card> cards;
    Deck() {
        for (auto& s : SUITS)
            for (auto& r : RANKS)
                cards.push_back(Card(s, r));
        shuffle();
    }
    void shuffle() {
        random_device rd;
        mt19937 g(rd());
        ::shuffle(cards.begin(), cards.end(), g);
    }
    Card draw() {
        if (cards.empty()) return Card("", "");
        Card c = cards.back();
        cards.pop_back();
        return c;
    }
};

class Hand {
public:
    vector<Card> cards;
    int bet;
    bool done;
    Hand() : bet(0), done(false) {}
    void add(Card c) { cards.push_back(c); }
    int value() const {
        int total = 0;
        int aces = 0;
        for (auto& c : cards) {
            total += c.value;
            if (c.rank == "A") aces++;
        }
        while (total > 21 && aces > 0) {
            total -= 10;
            aces--;
        }
        return total;
    }
    bool isBlackjack() const {
        return cards.size() == 2 && value() == 21;
    }
    bool isBust() const { return value() > 21; }
    bool canSplit() const {
        return cards.size() == 2 && cards[0].rank == cards[1].rank;
    }
    Hand split() {
        Hand h;
        h.cards.push_back(cards.back());
        cards.pop_back();
        return h;
    }
    string str() const {
        string s;
        for (auto& c : cards) s += c.str() + " ";
        return s;
    }
};

class Blackjack {
public:
    int balance;
    int bet;
    Deck deck;
    vector<Hand> player_hands;
    Hand dealer_hand;
    bool insurance;
    int insurance_paid;
    map<string, int> stats;
    string stats_file;

    Blackjack(int bal) : balance(bal), bet(0), insurance(false), insurance_paid(0) {
        stats_file = string(getenv("HOME")) + "/.blackjack_stats.json";
        loadStats();
    }

    void loadStats() {
        ifstream f(stats_file);
        if (!f) return;
        string line;
        while (getline(f, line)) {
            size_t p = line.find(':');
            if (p != string::npos) {
                string key = line.substr(0, p);
                int val = stoi(line.substr(p+1));
                stats[key] = val;
            }
        }
        if (stats.find("wins") == stats.end()) {
            stats["wins"] = 0; stats["losses"] = 0; stats["pushes"] = 0; stats["blackjacks"] = 0;
        }
    }

    void saveStats() {
        ofstream f(stats_file);
        if (f) {
            f << "wins:" << stats["wins"] << "\n";
            f << "losses:" << stats["losses"] << "\n";
            f << "pushes:" << stats["pushes"] << "\n";
            f << "blackjacks:" << stats["blackjacks"] << "\n";
        }
    }

    void displayStats() {
        cout << colorize("Статистика:", BOLD) << endl;
        cout << "  Побед: " << stats["wins"] << endl;
        cout << "  Поражений: " << stats["losses"] << endl;
        cout << "  Ничьих: " << stats["pushes"] << endl;
        cout << "  Блэкджеков: " << stats["blackjacks"] << endl;
        cout << "  Баланс: " << balance << endl;
    }

    void dealInitial() {
        deck = Deck();
        deck.shuffle();
        player_hands.clear();
        player_hands.push_back(Hand());
        dealer_hand = Hand();
        insurance = false;
        insurance_paid = 0;
        player_hands[0].add(deck.draw());
        dealer_hand.add(deck.draw());
        player_hands[0].add(deck.draw());
        dealer_hand.add(deck.draw());
    }

    void showHands(bool hideDealer = true) {
        cout << "\nДилер:" << endl;
        if (hideDealer) {
            cout << "  " << dealer_hand.cards[0].str() << "  [скрыто]" << endl;
        } else {
            cout << "  " << dealer_hand.str() << " (очки: " << dealer_hand.value() << ")" << endl;
        }
        cout << "\nВаши руки:" << endl;
        for (size_t i=0; i<player_hands.size(); ++i) {
            cout << "  Рука " << i+1 << ": " << player_hands[i].str() << " (очки: " << player_hands[i].value() << ")" << endl;
        }
    }

    void getBet() {
        while (true) {
            cout << "Ваша ставка (баланс: " << balance << "): ";
            string inp;
            cin >> inp;
            try {
                int b = stoi(inp);
                if (b > 0 && b <= balance) {
                    bet = b;
                    return;
                } else {
                    cout << colorize("Неверная ставка.", RED) << endl;
                }
            } catch (...) {
                cout << colorize("Введите число.", RED) << endl;
            }
        }
    }

    void playHand(int idx) {
        Hand& hand = player_hands[idx];
        while (!hand.done && !hand.isBust() && hand.value() < 21) {
            showHands();
            cout << "\nРука " << idx+1 << ": " << hand.str() << " (очки: " << hand.value() << ")" << endl;
            cout << "Действие (h=hit, s=stand, d=double, i=insurance, sp=split, q=quit): ";
            string action;
            cin >> action;
            if (action == "q") {
                cout << colorize("Выход.", YELLOW) << endl;
                saveStats();
                exit(0);
            } else if (action == "h") {
                hand.add(deck.draw());
                if (hand.isBust()) {
                    cout << colorize("Перебор!", RED) << endl;
                    hand.done = true;
                }
            } else if (action == "s") {
                hand.done = true;
            } else if (action == "d") {
                if (hand.cards.size() == 2 && balance >= bet) {
                    bet *= 2;
                    hand.add(deck.draw());
                    hand.done = true;
                    if (hand.isBust()) cout << colorize("Перебор!", RED) << endl;
                } else {
                    cout << colorize("Удвоение недоступно.", YELLOW) << endl;
                }
            } else if (action == "i") {
                if (dealer_hand.cards[0].rank == "A" && !insurance) {
                    insurance = true;
                    insurance_paid = bet / 2;
                    balance -= insurance_paid;
                    cout << colorize("Страховка активирована.", CYAN) << endl;
                } else {
                    cout << colorize("Страховка недоступна.", YELLOW) << endl;
                }
            } else if (action == "sp") {
                if (hand.canSplit()) {
                    Hand new_hand = hand.split();
                    player_hands.push_back(new_hand);
                    hand.add(deck.draw());
                    new_hand.add(deck.draw());
                    cout << colorize("Руки разделены.", CYAN) << endl;
                    if (balance < bet) {
                        cout << colorize("Недостаточно средств для сплита.", RED) << endl;
                        // откат
                        player_hands.pop_back();
                    } else {
                        balance -= bet;
                        bet *= 2;
                    }
                } else {
                    cout << colorize("Сплит недоступен.", YELLOW) << endl;
                }
            } else {
                cout << colorize("Неизвестная команда.", RED) << endl;
            }
        }
    }

    void dealerPlay() {
        while (dealer_hand.value() < 17) {
            dealer_hand.add(deck.draw());
        }
    }

    void resolve() {
        showHands(false);
        int dealer_val = dealer_hand.value();
        for (auto& hand : player_hands) {
            int player_val = hand.value();
            if (hand.isBust()) {
                balance -= bet;
                stats["losses"]++;
                cout << colorize("Поражение (перебор): рука проиграла " + to_string(bet), RED) << endl;
            } else if (hand.isBlackjack() && !dealer_hand.isBlackjack()) {
                int win = bet * 1.5;
                balance += win;
                stats["blackjacks"]++;
                stats["wins"]++;
                cout << colorize("Блэкджек! Выигрыш " + to_string(win), GREEN) << endl;
            } else if (dealer_val > 21) {
                balance += bet;
                stats["wins"]++;
                cout << colorize("Дилер перебрал, выигрыш " + to_string(bet), GREEN) << endl;
            } else if (player_val > dealer_val) {
                balance += bet;
                stats["wins"]++;
                cout << colorize("Выигрыш " + to_string(bet), GREEN) << endl;
            } else if (player_val == dealer_val) {
                stats["pushes"]++;
                cout << colorize("Ничья, ставка возвращена.", YELLOW) << endl;
            } else {
                balance -= bet;
                stats["losses"]++;
                cout << colorize("Поражение, потеряно " + to_string(bet), RED) << endl;
            }
        }
        if (insurance) {
            if (dealer_hand.isBlackjack()) {
                balance += insurance_paid * 2;
                cout << colorize("Страховка выиграла!", CYAN) << endl;
            } else {
                cout << colorize("Страховка проиграла.", YELLOW) << endl;
            }
        }
        saveStats();
    }

    void play() {
        cout << colorize("🃏 Добро пожаловать в Блэкджек!", BOLD) << endl;
        cout << "Ваш баланс: " << balance << endl;
        while (balance > 0) {
            getBet();
            dealInitial();
            // Блэкджек игрока
            if (player_hands[0].isBlackjack()) {
                showHands();
                if (dealer_hand.isBlackjack()) {
                    cout << colorize("Ничья – у обоих блэкджек!", YELLOW) << endl;
                    stats["pushes"]++;
                    saveStats();
                    continue;
                } else {
                    int win = bet * 1.5;
                    balance += win;
                    stats["blackjacks"]++;
                    stats["wins"]++;
                    cout << colorize("Блэкджек! Выигрыш " + to_string(win), GREEN) << endl;
                    saveStats();
                    continue;
                }
            }
            // Страховка
            if (dealer_hand.cards[0].rank == "A") {
                cout << "У дилера туз. Хотите страховку? (y/n): ";
                string ans;
                cin >> ans;
                if (ans == "y") {
                    insurance = true;
                    insurance_paid = bet / 2;
                    balance -= insurance_paid;
                    cout << colorize("Страховка активирована.", CYAN) << endl;
                }
            }
            // Ход игрока
            for (size_t i=0; i<player_hands.size(); ++i) {
                playHand(i);
            }
            // Ход дилера
            dealerPlay();
            resolve();
            cout << colorize("Баланс: " + to_string(balance), BOLD) << endl;
            if (balance <= 0) {
                cout << colorize("Вы проиграли все деньги!", RED) << endl;
                break;
            }
            cout << "Продолжить игру? (y/n): ";
            string cont;
            cin >> cont;
            if (cont != "y") break;
        }
        displayStats();
        saveStats();
    }
};

int main(int argc, char* argv[]) {
    int balance = 1000;
    bool reset = false;
    for (int i=1; i<argc; ++i) {
        string arg = argv[i];
        if (arg == "-r" || arg == "--reset") reset = true;
        else if ((arg == "-s" || arg == "--start") && i+1 < argc) {
            balance = stoi(argv[++i]);
        } else if (arg == "-h" || arg == "--help") {
            cout << "Usage: blackjack [-s start_balance] [-r]" << endl;
            return 0;
        }
    }
    if (reset) {
        string home = getenv("HOME");
        string f = home + "/.blackjack_stats.json";
        if (fs::exists(f)) fs::remove(f);
        cout << "Статистика сброшена." << endl;
        return 0;
    }
    Blackjack game(balance);
    game.play();
    return 0;
}
