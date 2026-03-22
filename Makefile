build:
	GOOS=wasip1 GOARCH=wasm go build -o chart.wasm ./main.go

test:
	go test -count=1 -timeout 30s ./...
