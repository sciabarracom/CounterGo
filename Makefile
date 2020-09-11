IMG=openwhisk/action-golang-v1.15:nightly

chess.zip: main.go
	find . -name \*.go -o -name go.\* |\
	xargs zip -r - |\
	docker run -i $(IMG) -compile main >$@

chess:
	go build -tags=cli -o chess
