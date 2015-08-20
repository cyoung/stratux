all:
	GOOS=linux GOARCH=arm GOARM=6 go build gen_gdl90.go traffic.go ry835ai.go 
clean:
	rm -f gen_gdl90
