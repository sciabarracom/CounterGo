module github.com/openwhisk-blog/nimbella-samples/CounterGo

go 1.15

replace github.com/ChizhovVadim/CounterGo/common => ./common

replace github.com/ChizhovVadim/CounterGo/eval => ./eval

replace github.com/ChizhovVadim/CounterGo/engine => ./engine

replace github.com/ChizhovVadim/CounterGo/uci => ./uci

require (
	github.com/ChizhovVadim/CounterGo/common v0.0.0-00010101000000-000000000000
	github.com/ChizhovVadim/CounterGo/engine v0.0.0-00010101000000-000000000000
	github.com/ChizhovVadim/CounterGo/eval v0.0.0-00010101000000-000000000000
	github.com/ChizhovVadim/CounterGo/uci v0.0.0-00010101000000-000000000000
)
