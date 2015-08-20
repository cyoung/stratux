all:
	GOOS=linux GOARCH=arm GOARM=6 go build gen_gdl90.go traffic.go ry835ai.go 
	GOOS=linux GOARCH=arm GOARM=6 go build 1090es_relay.go
clean:
	rm -f gen_gdl90
	rm -f 1090es_relay