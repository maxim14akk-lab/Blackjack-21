# blackjack.py
#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import sys
import os
import json
import random
from pathlib import Path

COLORS = {
    'reset': '\033[0m',
    'red': '\033[91m',
    'green': '\033[92m',
    'yellow': '\033[93m',
    'blue': '\033[94m',
    'cyan': '\033[96m',
    'bold': '\033[1m'
}

def colorize(text, color):
    return f"{COLORS.get(color, '')}{text}{COLORS['reset']}"

SUITS = ['♠', '♥', '♦', '♣']
RANKS = ['2', '3', '4', '5', '6', '7', '8', '9', '10', 'J', 'Q', 'K', 'A']
VALUES = {'2':2,'3':3,'4':4,'5':5,'6':6,'7':7,'8':8,'9':9,'10':10,'J':10,'Q':10,'K':10,'A':11}

class Card:
    def __init__(self, suit, rank):
        self.suit = suit
        self.rank = rank
        self.value = VALUES[rank]
        self.color = 'red' if suit in ('♥', '♦') else 'black'

    def __str__(self):
        return colorize(f"{self.rank}{self.suit}", self.color)

class Deck:
    def __init__(self):
        self.cards = [Card(s, r) for s in SUITS for r in RANKS]
        self.shuffle()

    def shuffle(self):
        random.shuffle(self.cards)

    def draw(self):
        return self.cards.pop() if self.cards else None

class Hand:
    def __init__(self, cards=None):
        self.cards = cards or []
        self.bet = 0
        self.is_done = False

    def add(self, card):
        self.cards.append(card)

    def value(self):
        total = sum(c.value for c in self.cards)
        aces = sum(1 for c in self.cards if c.rank == 'A')
        while total > 21 and aces:
            total -= 10
            aces -= 1
        return total

    def is_blackjack(self):
        return len(self.cards) == 2 and self.value() == 21

    def is_bust(self):
        return self.value() > 21

    def can_split(self):
        return len(self.cards) == 2 and self.cards[0].rank == self.cards[1].rank

    def split(self):
        return Hand([self.cards.pop()])

    def __str__(self):
        return ' '.join(str(c) for c in self.cards)

class Blackjack:
    def __init__(self, balance=1000):
        self.balance = balance
        self.bet = 0
        self.deck = Deck()
        self.player_hands = [Hand()]
        self.dealer_hand = Hand()
        self.insurance = False
        self.insurance_paid = 0
        self.stats_file = Path.home() / '.blackjack_stats.json'
        self.load_stats()

    def load_stats(self):
        if self.stats_file.exists():
            with open(self.stats_file, 'r') as f:
                self.stats = json.load(f)
        else:
            self.stats = {'wins': 0, 'losses': 0, 'pushes': 0, 'blackjacks': 0}

    def save_stats(self):
        with open(self.stats_file, 'w') as f:
            json.dump(self.stats, f)

    def display_stats(self):
        print(colorize("Статистика:", 'bold'))
        print(f"  Побед: {self.stats['wins']}")
        print(f"  Поражений: {self.stats['losses']}")
        print(f"  Ничьих: {self.stats['pushes']}")
        print(f"  Блэкджеков: {self.stats['blackjacks']}")
        print(f"  Баланс: {self.balance}")

    def deal_initial(self):
        self.deck = Deck()
        self.deck.shuffle()
        self.player_hands = [Hand()]
        self.dealer_hand = Hand()
        self.insurance = False
        self.insurance_paid = 0
        for _ in range(2):
            self.player_hands[0].add(self.deck.draw())
            self.dealer_hand.add(self.deck.draw())

    def show_hands(self, hide_dealer=True):
        print("\nДилер:")
        if hide_dealer:
            print(f"  {self.dealer_hand.cards[0]}  [скрыто]")
        else:
            print(f"  {self.dealer_hand}  (очки: {self.dealer_hand.value()})")
        print("\nВаши руки:")
        for i, hand in enumerate(self.player_hands):
            print(f"  Рука {i+1}: {hand}  (очки: {hand.value()})")

    def get_bet(self):
        while True:
            try:
                bet = int(input(f"Ваша ставка (баланс: {self.balance}): "))
                if 0 < bet <= self.balance:
                    self.bet = bet
                    return
                else:
                    print(colorize("Неверная ставка.", 'red'))
            except ValueError:
                print(colorize("Введите число.", 'red'))

    def play_hand(self, hand_idx):
        hand = self.player_hands[hand_idx]
        while not hand.is_done and not hand.is_bust() and hand.value() < 21:
            self.show_hands()
            print(f"\nРука {hand_idx+1}: {hand}  (очки: {hand.value()})")
            action = input("Действие (h=hit, s=stand, d=double, i=insurance, sp=split, q=quit): ").strip().lower()
            if action == 'q':
                print(colorize("Выход.", 'yellow'))
                self.save_stats()
                sys.exit(0)
            elif action == 'h':
                hand.add(self.deck.draw())
                if hand.is_bust():
                    print(colorize("Перебор!", 'red'))
                    hand.is_done = True
            elif action == 's':
                hand.is_done = True
            elif action == 'd':
                if len(hand.cards) == 2 and self.balance >= self.bet:
                    self.bet *= 2
                    hand.add(self.deck.draw())
                    hand.is_done = True
                    if hand.is_bust():
                        print(colorize("Перебор!", 'red'))
                else:
                    print(colorize("Удвоение недоступно.", 'yellow'))
            elif action == 'i':
                if self.dealer_hand.cards[0].rank == 'A' and not self.insurance:
                    self.insurance = True
                    self.insurance_paid = self.bet // 2
                    self.balance -= self.insurance_paid
                    print(colorize("Страховка активирована.", 'cyan'))
                else:
                    print(colorize("Страховка недоступна.", 'yellow'))
            elif action == 'sp':
                if hand.can_split():
                    new_hand = hand.split()
                    self.player_hands.append(new_hand)
                    # Добавляем по одной карте к каждой руке
                    hand.add(self.deck.draw())
                    new_hand.add(self.deck.draw())
                    print(colorize("Руки разделены.", 'cyan'))
                    # Проверяем, что баланс позволяет дополнительную ставку
                    if self.balance < self.bet:
                        print(colorize("Недостаточно средств для сплита.", 'red'))
                        # откат
                        self.player_hands.pop()
                        # возвращаем карту обратно
                        # Упрощённо: просто игнорируем
                    else:
                        self.balance -= self.bet  # вторая ставка
                        self.bet *= 2  # общая ставка = 2*bet
                else:
                    print(colorize("Сплит недоступен.", 'yellow'))
            else:
                print(colorize("Неизвестная команда.", 'red'))

    def dealer_play(self):
        while self.dealer_hand.value() < 17:
            self.dealer_hand.add(self.deck.draw())
        return self.dealer_hand.value()

    def resolve(self):
        self.show_hands(hide_dealer=False)
        dealer_value = self.dealer_hand.value()
        for hand in self.player_hands:
            player_value = hand.value()
            if hand.is_bust():
                self.balance -= self.bet
                self.stats['losses'] += 1
                print(colorize(f"Поражение (перебор): рука проиграла {self.bet}", 'red'))
            elif hand.is_blackjack() and not self.dealer_hand.is_blackjack():
                win = int(self.bet * 1.5)
                self.balance += win
                self.stats['blackjacks'] += 1
                self.stats['wins'] += 1
                print(colorize(f"Блэкджек! Выигрыш {win}", 'green'))
            elif dealer_value > 21:
                self.balance += self.bet
                self.stats['wins'] += 1
                print(colorize(f"Дилер перебрал, выигрыш {self.bet}", 'green'))
            elif player_value > dealer_value:
                self.balance += self.bet
                self.stats['wins'] += 1
                print(colorize(f"Выигрыш {self.bet}", 'green'))
            elif player_value == dealer_value:
                self.stats['pushes'] += 1
                print(colorize("Ничья, ставка возвращена.", 'yellow'))
            else:
                self.balance -= self.bet
                self.stats['losses'] += 1
                print(colorize(f"Поражение, потеряно {self.bet}", 'red'))
        # Страховка
        if self.insurance:
            if self.dealer_hand.is_blackjack():
                self.balance += self.insurance_paid * 2
                print(colorize("Страховка выиграла!", 'cyan'))
            else:
                print(colorize("Страховка проиграла.", 'yellow'))

        self.save_stats()

    def play(self):
        print(colorize("🃏 Добро пожаловать в Блэкджек!", 'bold'))
        print(f"Ваш баланс: {self.balance}")
        while self.balance > 0:
            self.get_bet()
            self.deal_initial()
            # Проверка блэкджека у игрока
            if self.player_hands[0].is_blackjack():
                self.show_hands()
                if self.dealer_hand.is_blackjack():
                    print(colorize("Ничья – у обоих блэкджек!", 'yellow'))
                    self.stats['pushes'] += 1
                    self.save_stats()
                    continue
                else:
                    win = int(self.bet * 1.5)
                    self.balance += win
                    self.stats['blackjacks'] += 1
                    self.stats['wins'] += 1
                    print(colorize(f"Блэкджек! Выигрыш {win}", 'green'))
                    self.save_stats()
                    continue
            # Проверка блэкджека у дилера (игрок может взять страховку)
            if self.dealer_hand.cards[0].rank == 'A':
                # Предлагаем страховку
                print("У дилера туз. Хотите страховку? (y/n): ", end='')
                ans = input().strip().lower()
                if ans == 'y':
                    self.insurance = True
                    self.insurance_paid = self.bet // 2
                    self.balance -= self.insurance_paid
                    print(colorize("Страховка активирована.", 'cyan'))
            # Ход игрока
            for i in range(len(self.player_hands)):
                self.play_hand(i)
            # Ход дилера
            self.dealer_play()
            self.resolve()
            print(colorize(f"Баланс: {self.balance}", 'bold'))
            if self.balance <= 0:
                print(colorize("Вы проиграли все деньги!", 'red'))
                break
            if input("Продолжить игру? (y/n): ").strip().lower() != 'y':
                break
        self.display_stats()
        self.save_stats()

def main():
    balance = 1000
    reset = False
    if len(sys.argv) > 1:
        if sys.argv[1] == '-r' or sys.argv[1] == '--reset':
            reset = True
        elif sys.argv[1] == '-s' and len(sys.argv) > 2:
            balance = int(sys.argv[2])
        elif sys.argv[1] == '-h' or sys.argv[1] == '--help':
            print("Usage: blackjack.py [-s start_balance] [-r]")
            return
    if reset:
        stats_file = Path.home() / '.blackjack_stats.json'
        if stats_file.exists():
            stats_file.unlink()
        print("Статистика сброшена.")
        return
    game = Blackjack(balance)
    game.play()

if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print(colorize("\nИгра прервана.", 'yellow'))
        sys.exit(0)
