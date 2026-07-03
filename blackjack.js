// blackjack.js
#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');
const readline = require('readline');

const COLORS = {
    reset: '\x1b[0m',
    red: '\x1b[91m',
    green: '\x1b[92m',
    yellow: '\x1b[93m',
    blue: '\x1b[94m',
    cyan: '\x1b[96m',
    bold: '\x1b[1m'
};

function colorize(text, color) {
    return COLORS[color] + text + COLORS.reset;
}

const SUITS = ['♠','♥','♦','♣'];
const RANKS = ['2','3','4','5','6','7','8','9','10','J','Q','K','A'];
const VALUES = {2:2,3:3,4:4,5:5,6:6,7:7,8:8,9:9,10:10,J:10,Q:10,K:10,A:11};

class Card {
    constructor(suit, rank) {
        this.suit = suit;
        this.rank = rank;
        this.value = VALUES[rank];
        this.color = (suit === '♥' || suit === '♦') ? 'red' : 'reset';
    }
    toString() {
        return colorize(this.rank + this.suit, this.color);
    }
}

class Deck {
    constructor() {
        this.cards = [];
        for (const s of SUITS)
            for (const r of RANKS)
                this.cards.push(new Card(s, r));
        this.shuffle();
    }
    shuffle() {
        for (let i = this.cards.length - 1; i > 0; i--) {
            const j = Math.floor(Math.random() * (i+1));
            [this.cards[i], this.cards[j]] = [this.cards[j], this.cards[i]];
        }
    }
    draw() {
        return this.cards.pop();
    }
}

class Hand {
    constructor() {
        this.cards = [];
        this.bet = 0;
        this.done = false;
    }
    add(card) { this.cards.push(card); }
    value() {
        let total = this.cards.reduce((s,c) => s + c.value, 0);
        let aces = this.cards.filter(c => c.rank === 'A').length;
        while (total > 21 && aces > 0) {
            total -= 10;
            aces--;
        }
        return total;
    }
    isBlackjack() { return this.cards.length === 2 && this.value() === 21; }
    isBust() { return this.value() > 21; }
    canSplit() { return this.cards.length === 2 && this.cards[0].rank === this.cards[1].rank; }
    split() {
        const h = new Hand();
        h.cards.push(this.cards.pop());
        return h;
    }
    toString() {
        return this.cards.map(c => c.toString()).join(' ');
    }
}

class Blackjack {
    constructor(balance = 1000) {
        this.balance = balance;
        this.bet = 0;
        this.deck = null;
        this.playerHands = [];
        this.dealerHand = null;
        this.insurance = false;
        this.insurancePaid = 0;
        this.statsFile = path.join(os.homedir(), '.blackjack_stats.json');
        this.loadStats();
    }

    loadStats() {
        try {
            this.stats = JSON.parse(fs.readFileSync(this.statsFile, 'utf8'));
        } catch {
            this.stats = { wins: 0, losses: 0, pushes: 0, blackjacks: 0 };
        }
    }

    saveStats() {
        fs.writeFileSync(this.statsFile, JSON.stringify(this.stats, null, 2));
    }

    displayStats() {
        console.log(colorize('Статистика:', 'bold'));
        console.log(`  Побед: ${this.stats.wins}`);
        console.log(`  Поражений: ${this.stats.losses}`);
        console.log(`  Ничьих: ${this.stats.pushes}`);
        console.log(`  Блэкджеков: ${this.stats.blackjacks}`);
        console.log(`  Баланс: ${this.balance}`);
    }

    dealInitial() {
        this.deck = new Deck();
        this.deck.shuffle();
        this.playerHands = [new Hand()];
        this.dealerHand = new Hand();
        this.insurance = false;
        this.insurancePaid = 0;
        this.playerHands[0].add(this.deck.draw());
        this.dealerHand.add(this.deck.draw());
        this.playerHands[0].add(this.deck.draw());
        this.dealerHand.add(this.deck.draw());
    }

    showHands(hideDealer = true) {
        console.log('\nДилер:');
        if (hideDealer) {
            console.log(`  ${this.dealerHand.cards[0].toString()}  [скрыто]`);
        } else {
            console.log(`  ${this.dealerHand.toString()} (очки: ${this.dealerHand.value()})`);
        }
        console.log('\nВаши руки:');
        for (let i=0; i<this.playerHands.length; i++) {
            console.log(`  Рука ${i+1}: ${this.playerHands[i].toString()} (очки: ${this.playerHands[i].value()})`);
        }
    }

    async getBet() {
        const rl = readline.createInterface({
            input: process.stdin,
            output: process.stdout
        });
        while (true) {
            const ans = await new Promise(resolve => rl.question(`Ваша ставка (баланс: ${this.balance}): `, resolve));
            const bet = parseInt(ans);
            if (!isNaN(bet) && bet > 0 && bet <= this.balance) {
                this.bet = bet;
                rl.close();
                return;
            }
            console.log(colorize('Неверная ставка.', 'red'));
        }
    }

    async playHand(idx) {
        const hand = this.playerHands[idx];
        const rl = readline.createInterface({
            input: process.stdin,
            output: process.stdout
        });
        while (!hand.done && !hand.isBust() && hand.value() < 21) {
            this.showHands();
            console.log(`\nРука ${idx+1}: ${hand.toString()} (очки: ${hand.value()})`);
            const action = await new Promise(resolve => rl.question('Действие (h=hit, s=stand, d=double, i=insurance, sp=split, q=quit): ', resolve));
            const cmd = action.trim().toLowerCase();
            if (cmd === 'q') {
                console.log(colorize('Выход.', 'yellow'));
                this.saveStats();
                process.exit(0);
            } else if (cmd === 'h') {
                hand.add(this.deck.draw());
                if (hand.isBust()) {
                    console.log(colorize('Перебор!', 'red'));
                    hand.done = true;
                }
            } else if (cmd === 's') {
                hand.done = true;
            } else if (cmd === 'd') {
                if (hand.cards.length === 2 && this.balance >= this.bet) {
                    this.bet *= 2;
                    hand.add(this.deck.draw());
                    hand.done = true;
                    if (hand.isBust()) console.log(colorize('Перебор!', 'red'));
                } else {
                    console.log(colorize('Удвоение недоступно.', 'yellow'));
                }
            } else if (cmd === 'i') {
                if (this.dealerHand.cards[0].rank === 'A' && !this.insurance) {
                    this.insurance = true;
                    this.insurancePaid = Math.floor(this.bet / 2);
                    this.balance -= this.insurancePaid;
                    console.log(colorize('Страховка активирована.', 'cyan'));
                } else {
                    console.log(colorize('Страховка недоступна.', 'yellow'));
                }
            } else if (cmd === 'sp') {
                if (hand.canSplit()) {
                    const newHand = hand.split();
                    this.playerHands.push(newHand);
                    hand.add(this.deck.draw());
                    newHand.add(this.deck.draw());
                    console.log(colorize('Руки разделены.', 'cyan'));
                    if (this.balance < this.bet) {
                        console.log(colorize('Недостаточно средств для сплита.', 'red'));
                        this.playerHands.pop();
                    } else {
                        this.balance -= this.bet;
                        this.bet *= 2;
                    }
                } else {
                    console.log(colorize('Сплит недоступен.', 'yellow'));
                }
            } else {
                console.log(colorize('Неизвестная команда.', 'red'));
            }
        }
        rl.close();
    }

    dealerPlay() {
        while (this.dealerHand.value() < 17) {
            this.dealerHand.add(this.deck.draw());
        }
    }

    resolve() {
        this.showHands(false);
        const dealerVal = this.dealerHand.value();
        for (const hand of this.playerHands) {
            const playerVal = hand.value();
            if (hand.isBust()) {
                this.balance -= this.bet;
                this.stats.losses++;
                console.log(colorize(`Поражение (перебор): рука проиграла ${this.bet}`, 'red'));
            } else if (hand.isBlackjack() && !this.dealerHand.isBlackjack()) {
                const win = Math.floor(this.bet * 1.5);
                this.balance += win;
                this.stats.blackjacks++;
                this.stats.wins++;
                console.log(colorize(`Блэкджек! Выигрыш ${win}`, 'green'));
            } else if (dealerVal > 21) {
                this.balance += this.bet;
                this.stats.wins++;
                console.log(colorize(`Дилер перебрал, выигрыш ${this.bet}`, 'green'));
            } else if (playerVal > dealerVal) {
                this.balance += this.bet;
                this.stats.wins++;
                console.log(colorize(`Выигрыш ${this.bet}`, 'green'));
            } else if (playerVal === dealerVal) {
                this.stats.pushes++;
                console.log(colorize('Ничья, ставка возвращена.', 'yellow'));
            } else {
                this.balance -= this.bet;
                this.stats.losses++;
                console.log(colorize(`Поражение, потеряно ${this.bet}`, 'red'));
            }
        }
        if (this.insurance) {
            if (this.dealerHand.isBlackjack()) {
                this.balance += this.insurancePaid * 2;
                console.log(colorize('Страховка выиграла!', 'cyan'));
            } else {
                console.log(colorize('Страховка проиграла.', 'yellow'));
            }
        }
        this.saveStats();
    }

    async play() {
        console.log(colorize('🃏 Добро пожаловать в Блэкджек!', 'bold'));
        console.log(`Ваш баланс: ${this.balance}`);
        while (this.balance > 0) {
            await this.getBet();
            this.dealInitial();
            // Блэкджек игрока
            if (this.playerHands[0].isBlackjack()) {
                this.showHands();
                if (this.dealerHand.isBlackjack()) {
                    console.log(colorize('Ничья – у обоих блэкджек!', 'yellow'));
                    this.stats.pushes++;
                    this.saveStats();
                    continue;
                } else {
                    const win = Math.floor(this.bet * 1.5);
                    this.balance += win;
                    this.stats.blackjacks++;
                    this.stats.wins++;
                    console.log(colorize(`Блэкджек! Выигрыш ${win}`, 'green'));
                    this.saveStats();
                    continue;
                }
            }
            // Страховка
            if (this.dealerHand.cards[0].rank === 'A') {
                const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
                const ans = await new Promise(resolve => rl.question('У дилера туз. Хотите страховку? (y/n): ', resolve));
                rl.close();
                if (ans.trim().toLowerCase() === 'y') {
                    this.insurance = true;
                    this.insurancePaid = Math.floor(this.bet / 2);
                    this.balance -= this.insurancePaid;
                    console.log(colorize('Страховка активирована.', 'cyan'));
                }
            }
            // Ход игрока
            for (let i=0; i<this.playerHands.length; i++) {
                await this.playHand(i);
            }
            // Ход дилера
            this.dealerPlay();
            this.resolve();
            console.log(colorize(`Баланс: ${this.balance}`, 'bold'));
            if (this.balance <= 0) {
                console.log(colorize('Вы проиграли все деньги!', 'red'));
                break;
            }
            const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
            const cont = await new Promise(resolve => rl.question('Продолжить игру? (y/n): ', resolve));
            rl.close();
            if (cont.trim().toLowerCase() !== 'y') break;
        }
        this.displayStats();
        this.saveStats();
    }
}

async function main() {
    let balance = 1000;
    let reset = false;
    for (let i=2; i<process.argv.length; i++) {
        const arg = process.argv[i];
        if (arg === '-r' || arg === '--reset') reset = true;
        else if ((arg === '-s' || arg === '--start') && i+1 < process.argv.length) {
            balance = parseInt(process.argv[++i]);
        } else if (arg === '-h' || arg === '--help') {
            console.log('Usage: node blackjack.js [-s start_balance] [-r]');
            process.exit(0);
        }
    }
    if (reset) {
        const f = path.join(os.homedir(), '.blackjack_stats.json');
        if (fs.existsSync(f)) fs.unlinkSync(f);
        console.log('Статистика сброшена.');
        return;
    }
    const game = new Blackjack(balance);
    await game.play();
}

main().catch(console.error);
