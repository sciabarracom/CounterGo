IMG=openwhisk/action-golang-v1.15:nightly

chess.zip: main.go index.go
	find . -name \*.go -o -name go.\* |\
	xargs zip -r - |\
	docker run -i $(IMG) -compile main >$@

chess: cli.go chess.zip
	go build -tags=cli -o chess

index.go: index.html
	echo "package main" >$@
	echo 'var indexHTML = `<!DOCTYPE html>' >>$@
	grep -v REMOVE index.html >>$@
	echo '`;' >>$@

deploy: chess.zip
	nim action update chess chess.zip \
	--docker openwhisk/action-golang-v1.15:nightly --web true
	nim action get chess --url

clean:
	-rm chess chess.zip
