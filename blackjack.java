// blackjack.java
import java.io.*;
import java.nio.file.*;
import java.util.*;

public class blackjack {
    private static final String RESET = "\u001B[0m";
    private static final String RED = "\u001B[91m";
    private static final String GREEN = "\u001B[92m";
    private static final String YELLOW = "\u001B[93m";
    private static final String BLUE = "\u001B[94m";
    private static final String CYAN = "\u001B[96m";
    private static final String BOLD = "\u001B[1m";

    private static String colorize(String text, String color) {
        return color + text + RESET;
    }

    private static final String[] SUITS = {"♠","♥","♦","♣"};
    private static final String[] RANKS = {"2","3","4","5","6","7","8","9","10","J","Q","K","A"};
    private static final Map<String, Integer> VALUES = new HashMap<>();
    static {
        VALUES.put("2",2); VALUES.put("3",3); VALUES.put("4",4); VALUES.put("5",5);
        VALUES.put("6",6); VALUES.put("7",7); VALUES.put("8",8); VALUES.put("9",9);
        VALUES.put("10",10); VALUES.put("J",10); VALUES.put("Q",10); VALUES.put("K",10); VALUES.put("A",11);
    }

    static class Card {
        String suit, rank;
        int value;
        Card(String s, String r) { suit=s; rank=r; value=VALUES.get(r); }
        @Override
        public String toString() {
            String col = (suit.equals("♥") || suit.equals("♦")) ? RED : RESET;
            return colorize(rank + suit, col);
        }
    }

    static class Deck {
        List<Card> cards;
        Deck() {
            cards = new ArrayList<>();
            for (String s : SUITS) for (String r : RANKS) cards.add(new Card(s, r));
            shuffle();
        }
        void shuffle() {
            Collections.shuffle(cards);
        }
        Card draw() {
            if (cards.isEmpty()) return null;
            return cards.remove(cards.size()-1);
        }
    }

    static class Hand {
        List<Card> cards = new ArrayList<>();
        int bet;
        boolean done;
        int value() {
            int total = cards.stream().mapToInt(c -> c.value).sum();
            int aces = (int) cards.stream().filter(c -> c.rank.equals("A")).count();
            while (total > 21 && aces > 0) { total -= 10; aces--; }
            return total;
        }
        boolean isBlackjack() { return cards.size() == 2 && value() == 21; }
        boolean isBust() { return value() > 21; }
        boolean canSplit() { return cards.size() == 2 && cards.get(0).rank.equals(cards.get(1).rank); }
        Hand split() {
            Hand h = new Hand();
            h.cards.add(cards.remove(cards.size()-1));
            return h;
        }
        @Override
        public String toString() {
            StringBuilder sb = new StringBuilder();
            for (Card c : cards) sb.append(c.toString()).append(" ");
            return sb.toString().trim();
        }
    }

    static class Stats {
        int wins, losses, pushes, blackjacks;
    }

    static class Blackjack {
        int balance, bet;
        Deck deck;
        List<Hand> playerHands;
        Hand dealerHand;
        boolean insurance;
        int insurancePaid;
        Stats stats = new Stats();
        String statsFile;

        Blackjack(int bal) {
            balance = bal;
            statsFile = System.getProperty("user.home") + "/.blackjack_stats.json";
            loadStats();
        }

        void loadStats() {
            try {
                String json = new String(Files.readAllBytes(Paths.get(statsFile)));
                // упрощённый парсинг (без библиотек)
                stats.wins = extractInt(json, "wins");
                stats.losses = extractInt(json, "losses");
                stats.pushes = extractInt(json, "pushes");
                stats.blackjacks = extractInt(json, "blackjacks");
            } catch (Exception e) {}
        }

        int extractInt(String json, String key) {
            int idx = json.indexOf("\"" + key + "\":");
            if (idx == -1) return 0;
            int start = idx + key.length() + 3;
            int end = json.indexOf(",", start);
            if (end == -1) end = json.indexOf("}", start);
            try { return Integer.parseInt(json.substring(start, end).trim()); } catch (Exception e) { return 0; }
        }

        void saveStats() {
            try {
                String json = "{\"wins\":"+stats.wins+",\"losses\":"+stats.losses+
                              ",\"pushes\":"+stats.pushes+",\"blackjacks\":"+stats.blackjacks+"}";
                Files.write(Paths.get(statsFile), json.getBytes());
            } catch (Exception e) {}
        }

        void displayStats() {
            System.out.println(colorize("Статистика:", BOLD));
            System.out.println("  Побед: " + stats.wins);
            System.out.println("  Поражений: " + stats.losses);
            System.out.println("  Ничьих: " + stats.pushes);
            System.out.println("  Блэкджеков: " + stats.blackjacks);
            System.out.println("  Баланс: " + balance);
        }

        void dealInitial() {
            deck = new Deck();
            deck.shuffle();
            playerHands = new ArrayList<>();
            playerHands.add(new Hand());
            dealerHand = new Hand();
            insurance = false;
            insurancePaid = 0;
            playerHands.get(0).cards.add(deck.draw());
            dealerHand.cards.add(deck.draw());
            playerHands.get(0).cards.add(deck.draw());
            dealerHand.cards.add(deck.draw());
        }

        void showHands(boolean hideDealer) {
            System.out.println("\nДилер:");
            if (hideDealer) {
                System.out.println("  " + dealerHand.cards.get(0).toString() + "  [скрыто]");
            } else {
                System.out.println("  " + dealerHand.toString() + " (очки: " + dealerHand.value() + ")");
            }
            System.out.println("\nВаши руки:");
            for (int i=0; i<playerHands.size(); i++) {
                System.out.println("  Рука " + (i+1) + ": " + playerHands.get(i).toString() + " (очки: " + playerHands.get(i).value() + ")");
            }
        }

        void getBet() {
            Scanner sc = new Scanner(System.in);
            while (true) {
                System.out.print("Ваша ставка (баланс: " + balance + "): ");
                String inp = sc.nextLine();
                try {
                    int b = Integer.parseInt(inp);
                    if (b > 0 && b <= balance) { bet = b; return; }
                } catch (Exception e) {}
                System.out.println(colorize("Неверная ставка.", RED));
            }
        }

        void playHand(int idx, Scanner sc) {
            Hand hand = playerHands.get(idx);
            while (!hand.done && !hand.isBust() && hand.value() < 21) {
                showHands(true);
                System.out.println("\nРука " + (idx+1) + ": " + hand.toString() + " (очки: " + hand.value() + ")");
                System.out.print("Действие (h=hit, s=stand, d=double, i=insurance, sp=split, q=quit): ");
                String action = sc.nextLine().trim().toLowerCase();
                switch (action) {
                    case "q": System.out.println(colorize("Выход.", YELLOW)); saveStats(); System.exit(0); break;
                    case "h":
                        hand.cards.add(deck.draw());
                        if (hand.isBust()) { System.out.println(colorize("Перебор!", RED)); hand.done = true; }
                        break;
                    case "s": hand.done = true; break;
                    case "d":
                        if (hand.cards.size() == 2 && balance >= bet) {
                            bet *= 2;
                            hand.cards.add(deck.draw());
                            hand.done = true;
                            if (hand.isBust()) System.out.println(colorize("Перебор!", RED));
                        } else System.out.println(colorize("Удвоение недоступно.", YELLOW));
                        break;
                    case "i":
                        if (dealerHand.cards.get(0).rank.equals("A") && !insurance) {
                            insurance = true;
                            insurancePaid = bet / 2;
                            balance -= insurancePaid;
                            System.out.println(colorize("Страховка активирована.", CYAN));
                        } else System.out.println(colorize("Страховка недоступна.", YELLOW));
                        break;
                    case "sp":
                        if (hand.canSplit()) {
                            Hand newHand = hand.split();
                            playerHands.add(newHand);
                            hand.cards.add(deck.draw());
                            newHand.cards.add(deck.draw());
                            System.out.println(colorize("Руки разделены.", CYAN));
                            if (balance < bet) {
                                System.out.println(colorize("Недостаточно средств для сплита.", RED));
                                playerHands.remove(playerHands.size()-1);
                            } else {
                                balance -= bet;
                                bet *= 2;
                            }
                        } else System.out.println(colorize("Сплит недоступен.", YELLOW));
                        break;
                    default: System.out.println(colorize("Неизвестная команда.", RED));
                }
            }
        }

        void dealerPlay() {
            while (dealerHand.value() < 17) {
                dealerHand.cards.add(deck.draw());
            }
        }

        void resolve() {
            showHands(false);
            int dealerVal = dealerHand.value();
            for (Hand hand : playerHands) {
                int playerVal = hand.value();
                if (hand.isBust()) {
                    balance -= bet;
                    stats.losses++;
                    System.out.println(colorize("Поражение (перебор): рука проиграла " + bet, RED));
                } else if (hand.isBlackjack() && !dealerHand.isBlackjack()) {
                    int win = (int)(bet * 1.5);
                    balance += win;
                    stats.blackjacks++;
                    stats.wins++;
                    System.out.println(colorize("Блэкджек! Выигрыш " + win, GREEN));
                } else if (dealerVal > 21) {
                    balance += bet;
                    stats.wins++;
                    System.out.println(colorize("Дилер перебрал, выигрыш " + bet, GREEN));
                } else if (playerVal > dealerVal) {
                    balance += bet;
                    stats.wins++;
                    System.out.println(colorize("Выигрыш " + bet, GREEN));
                } else if (playerVal == dealerVal) {
                    stats.pushes++;
                    System.out.println(colorize("Ничья, ставка возвращена.", YELLOW));
                } else {
                    balance -= bet;
                    stats.losses++;
                    System.out.println(colorize("Поражение, потеряно " + bet, RED));
                }
            }
            if (insurance) {
                if (dealerHand.isBlackjack()) {
                    balance += insurancePaid * 2;
                    System.out.println(colorize("Страховка выиграла!", CYAN));
                } else System.out.println(colorize("Страховка проиграла.", YELLOW));
            }
            saveStats();
        }

        void play() {
            Scanner sc = new Scanner(System.in);
            System.out.println(colorize("🃏 Добро пожаловать в Блэкджек!", BOLD));
            System.out.println("Ваш баланс: " + balance);
            while (balance > 0) {
                getBet();
                dealInitial();
                if (playerHands.get(0).isBlackjack()) {
                    showHands(true);
                    if (dealerHand.isBlackjack()) {
                        System.out.println(colorize("Ничья – у обоих блэкджек!", YELLOW));
                        stats.pushes++;
                        saveStats();
                        continue;
                    } else {
                        int win = (int)(bet * 1.5);
                        balance += win;
                        stats.blackjacks++;
                        stats.wins++;
                        System.out.println(colorize("Блэкджек! Выигрыш " + win, GREEN));
                        saveStats();
                        continue;
                    }
                }
                if (dealerHand.cards.get(0).rank.equals("A")) {
                    System.out.print("У дилера туз. Хотите страховку? (y/n): ");
                    String ans = sc.nextLine().trim().toLowerCase();
                    if (ans.equals("y")) {
                        insurance = true;
                        insurancePaid = bet / 2;
                        balance -= insurancePaid;
                        System.out.println(colorize("Страховка активирована.", CYAN));
                    }
                }
                for (int i=0; i<playerHands.size(); i++) {
                    playHand(i, sc);
                }
                dealerPlay();
                resolve();
                System.out.println(colorize("Баланс: " + balance, BOLD));
                if (balance <= 0) {
                    System.out.println(colorize("Вы проиграли все деньги!", RED));
                    break;
                }
                System.out.print("Продолжить игру? (y/n): ");
                String cont = sc.nextLine().trim().toLowerCase();
                if (!cont.equals("y")) break;
            }
            displayStats();
            saveStats();
            sc.close();
        }
    }

    public static void main(String[] args) {
        int balance = 1000;
        boolean reset = false;
        for (int i=0; i<args.length; i++) {
            if (args[i].equals("-r") || args[i].equals("--reset")) reset = true;
            else if ((args[i].equals("-s") || args[i].equals("--start")) && i+1 < args.length) {
                balance = Integer.parseInt(args[++i]);
            } else if (args[i].equals("-h") || args[i].equals("--help")) {
                System.out.println("Usage: java blackjack [-s start_balance] [-r]");
                return;
            }
        }
        if (reset) {
            String f = System.getProperty("user.home") + "/.blackjack_stats.json";
            try { Files.deleteIfExists(Paths.get(f)); } catch (Exception e) {}
            System.out.println("Статистика сброшена.");
            return;
        }
        Blackjack game = new Blackjack(balance);
        game.play();
    }
}
