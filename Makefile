build:
	@go build -o bin/balancer main.go
exe:
	@GOOS=windows GOARCH=amd64 go build -o bin/balancer.exe main.go 
run: build
	@./bin/balancer