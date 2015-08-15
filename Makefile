all:
	GOARCH=6 go-linux-arm build gen_gdl90.go traffic.go ry835ai.go
clean:
	rm -f gen_gdl90