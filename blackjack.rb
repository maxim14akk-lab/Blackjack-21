#!/usr/bin/env ruby
# blackjack.rb
# encoding: UTF-8

require 'json'
require 'fileutils'

COLORS = {
  reset: "\e[0m",
  red: "\e[91m",
  green: "\e[92m",
  yellow: "\e[93m",
  blue: "\e[94m",
  cyan: "\e[96m",
  bold: "\e[1m"
}

def colorize(text, color)
  "#{COLORS[color]}#{text}#{COLORS[:reset]}"
end

SUITS = ['♠', '♥', '♦', '♣']
RANKS = ['2','3','4','5','6','7','8','9','10','J','Q','K','A']
VALUES = {'2'=>2,'3'=>3,'4'=>4,'5'=>5,'6'=>6,'7'=>7,'8'=>8,'9'=>9,'10'=>10,'J'=>10,'Q'=>10,'K'=>10,'A'=>11}

class Card
  attr_reader :suit, :rank, :value
  def initialize(suit, rank)
    @suit = suit
    @rank = rank
    @value = VALUES[rank]
  end
  def to_s
    color = (@suit == '♥' || @suit == '♦') ? :red : :reset
    colorize(@rank + @suit, color)
  end
end

class Deck
  attr_reader :cards
  def initialize
    @cards = []
    SUITS.each { |s| RANKS.each { |r| @cards << Card.new(s, r) } }
    shuffle
  end
  def shuffle
    @cards.shuffle!
  end
  def draw
    @cards.pop
  end
end

class Hand
  attr_accessor :cards, :bet, :done
  def initialize
    @cards = []
    @bet = 0
    @done = false
  end
  def value
    total = @cards.sum(&:value)
    aces = @cards.count { |c| c.rank == 'A' }
    while total > 21 && aces > 0
      total -= 10
      aces -= 1
    end
    total
  end
  def blackjack?
    @cards.size == 2 && value == 21
  end
  def bust?
    value > 21
  end
  def can_split?
    @cards.size == 2 && @cards[0].rank == @cards[1].rank
  end
  def split
    h = Hand.new
    h.cards << @cards.pop
    h
  end
  def to_s
    @cards.map(&:to_s).join(' ')
  end
end

class Blackjack
  attr_reader :balance, :stats_file

  def initialize(balance)
    @balance = balance
    @stats_file = File.join(Dir.home, '.blackjack_stats.json')
    load_stats
  end

  def load_stats
    if File.exist?(@stats_file)
      @stats = JSON.parse(File.read(@stats_file))
    else
      @stats = { 'wins' => 0, 'losses' => 0, 'pushes' => 0, 'blackjacks' => 0 }
    end
  end

  def save_stats
    File.write(@stats_file, JSON.pretty_generate(@stats))
  end

  def display_stats
    puts colorize("Статистика:", :bold)
    puts "  Побед: #{@stats['wins']}"
    puts "  Поражений: #{@stats['losses']}"
    puts "  Ничьих: #{@stats['pushes']}"
    puts "  Блэкджеков: #{@stats['blackjacks']}"
    puts "  Баланс: #{@balance}"
  end

  def deal_initial
    @deck = Deck.new
    @deck.shuffle
    @player_hands = [Hand.new]
    @dealer_hand = Hand.new
    @insurance = false
    @insurance_paid = 0
    @player_hands[0].cards << @deck.draw
    @dealer_hand.cards << @deck.draw
    @player_hands[0].cards << @deck.draw
    @dealer_hand.cards << @deck.draw
  end

  def show_hands(hide_dealer = true)
    puts "\nДилер:"
    if hide_dealer
      puts "  #{@dealer_hand.cards[0]}  [скрыто]"
    else
      puts "  #{@dealer_hand} (очки: #{@dealer_hand.value})"
    end
    puts "\nВаши руки:"
    @player_hands.each_with_index do |h, i|
      puts "  Рука #{i+1}: #{h} (очки: #{h.value})"
    end
  end

  def get_bet
    loop do
      print "Ваша ставка (баланс: #{@balance}): "
      input = gets.chomp
      bet = input.to_i
      if bet > 0 && bet <= @balance
        @bet = bet
        return
      end
      puts colorize("Неверная ставка.", :red)
    end
  end

  def play_hand(idx)
    hand = @player_hands[idx]
    while !hand.done && !hand.bust? && hand.value < 21
      show_hands
      puts "\nРука #{idx+1}: #{hand} (очки: #{hand.value})"
      print "Действие (h=hit, s=stand, d=double, i=insurance, sp=split, q=quit): "
      action = gets.chomp.strip.downcase
      case action
      when 'q'
        puts colorize("Выход.", :yellow)
        save_stats
        exit
      when 'h'
        hand.cards << @deck.draw
        if hand.bust?
          puts colorize("Перебор!", :red)
          hand.done = true
        end
      when 's'
        hand.done = true
      when 'd'
        if hand.cards.size == 2 && @balance >= @bet
          @bet *= 2
          hand.cards << @deck.draw
          hand.done = true
          puts colorize("Перебор!", :red) if hand.bust?
        else
          puts colorize("Удвоение недоступно.", :yellow)
        end
      when 'i'
        if @dealer_hand.cards[0].rank == 'A' && !@insurance
          @insurance = true
          @insurance_paid = @bet / 2
          @balance -= @insurance_paid
          puts colorize("Страховка активирована.", :cyan)
        else
          puts colorize("Страховка недоступна.", :yellow)
        end
      when 'sp'
        if hand.can_split?
          new_hand = hand.split
          @player_hands << new_hand
          hand.cards << @deck.draw
          new_hand.cards << @deck.draw
          puts colorize("Руки разделены.", :cyan)
          if @balance < @bet
            puts colorize("Недостаточно средств для сплита.", :red)
            @player_hands.pop
          else
            @balance -= @bet
            @bet *= 2
          end
        else
          puts colorize("Сплит недоступен.", :yellow)
        end
      else
        puts colorize("Неизвестная команда.", :red)
      end
    end
  end

  def dealer_play
    while @dealer_hand.value < 17
      @dealer_hand.cards << @deck.draw
    end
  end

  def resolve
    show_hands(false)
    dealer_val = @dealer_hand.value
    @player_hands.each do |hand|
      player_val = hand.value
      if hand.bust?
        @balance -= @bet
        @stats['losses'] += 1
        puts colorize("Поражение (перебор): рука проиграла #{@bet}", :red)
      elsif hand.blackjack? && !@dealer_hand.blackjack?
        win = (@bet * 1.5).to_i
        @balance += win
        @stats['blackjacks'] += 1
        @stats['wins'] += 1
        puts colorize("Блэкджек! Выигрыш #{win}", :green)
      elsif dealer_val > 21
        @balance += @bet
        @stats['wins'] += 1
        puts colorize("Дилер перебрал, выигрыш #{@bet}", :green)
      elsif player_val > dealer_val
        @balance += @bet
        @stats['wins'] += 1
        puts colorize("Выигрыш #{@bet}", :green)
      elsif player_val == dealer_val
        @stats['pushes'] += 1
        puts colorize("Ничья, ставка возвращена.", :yellow)
      else
        @balance -= @bet
        @stats['losses'] += 1
        puts colorize("Поражение, потеряно #{@bet}", :red)
      end
    end
    if @insurance
      if @dealer_hand.blackjack?
        @balance += @insurance_paid * 2
        puts colorize("Страховка выиграла!", :cyan)
      else
        puts colorize("Страховка проиграла.", :yellow)
      end
    end
    save_stats
  end

  def play
    puts colorize("🃏 Добро пожаловать в Блэкджек!", :bold)
    puts "Ваш баланс: #{@balance}"
    while @balance > 0
      get_bet
      deal_initial
      if @player_hands[0].blackjack?
        show_hands
        if @dealer_hand.blackjack?
          puts colorize("Ничья – у обоих блэкджек!", :yellow)
          @stats['pushes'] += 1
          save_stats
          next
        else
          win = (@bet * 1.5).to_i
          @balance += win
          @stats['blackjacks'] += 1
          @stats['wins'] += 1
          puts colorize("Блэкджек! Выигрыш #{win}", :green)
          save_stats
          next
        end
      end
      if @dealer_hand.cards[0].rank == 'A'
        print "У дилера туз. Хотите страховку? (y/n): "
        ans = gets.chomp.strip.downcase
        if ans == 'y'
          @insurance = true
          @insurance_paid = @bet / 2
          @balance -= @insurance_paid
          puts colorize("Страховка активирована.", :cyan)
        end
      end
      @player_hands.each_with_index { |_, i| play_hand(i) }
      dealer_play
      resolve
      puts colorize("Баланс: #{@balance}", :bold)
      if @balance <= 0
        puts colorize("Вы проиграли все деньги!", :red)
        break
      end
      print "Продолжить игру? (y/n): "
      cont = gets.chomp.strip.downcase
      break if cont != 'y'
    end
    display_stats
    save_stats
  end
end

def main
  balance = 1000
  reset = false
  i = 1
  while i < ARGV.size
    arg = ARGV[i]
    case arg
    when '-r', '--reset' then reset = true
    when '-s', '--start'
      balance = ARGV[i+1].to_i if i+1 < ARGV.size
      i += 1
    when '-h', '--help'
      puts "Usage: ruby blackjack.rb [-s start_balance] [-r]"
      return
    end
    i += 1
  end
  if reset
    f = File.join(Dir.home, '.blackjack_stats.json')
    File.delete(f) if File.exist?(f)
    puts "Статистика сброшена."
    return
  end
  game = Blackjack.new(balance)
  game.play
end

main if __FILE__ == $0
