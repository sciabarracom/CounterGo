# Counter
Counter is a free, open-source chess engine, implemented in [Go](https://golang.org/).
Counter supports standard UCI (universal chess interface) protocol.

## Strength

Chess ratings are available at Computer Chess Rating Lists:
- [CCRL 40/4](http://www.computerchess.org.uk/ccrl/404/cgi/compare_engines.cgi?family=Counter&print=Rating+list&print=Results+table&print=LOS+table&print=Ponder+hit+table&print=Eval+difference+table&print=Comopp+gamenum+table&print=Overlap+table&print=Score+with+common+opponents)

- [CCRL 40/40](http://www.computerchess.org.uk/ccrl/4040/cgi/compare_engines.cgi?family=Counter&print=Rating+list&print=Results+table&print=LOS+table&print=Ponder+hit+table&print=Eval+difference+table&print=Comopp+gamenum+table&print=Overlap+table&print=Score+with+common+opponents)

## Commands
Counter supports [UCI protocol](http://www.shredderchess.com/chess-info/features/uci-universal-chess-interface.html) commands and own commands:
+ `move e2e4` - play chess with engine in REPL mode

## Features
### Board
+ Magic bitboards
### Evaluation
+ Texel's Tuning Method
### Search
+ Parallel search
+ Iterative Deepening
+ Principal Variation Search
+ Transposition Table
+ Null Move Pruning
+ Late Move Reductions
+ Futility Pruning
+ Move Count Based Pruning

## Information about chess programming
+ [Chess Programming Wiki](https://www.chessprogramming.org)
+ [Bruce Moreland's Programming Topics](https://web.archive.org/web/20071026090003/http://www.brucemo.com/compchess/programming/index.htm)

## Thanks
+ Vladimir Medvedev, GreKo
+ Fabien Letouzey, Fruit and Senpai
+ Robert Hyatt, Crafty

---------------------------------------------------------------

Counter Copyright (c) Vadim Chizhov. All rights reserved.
