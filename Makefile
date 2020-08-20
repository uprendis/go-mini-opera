# build
.PHONY : build txstorm
build :
	go build -o build/miniopera ./cmd/miniopera

#test
.PHONY : test
test :
	go test ./...

#clean
.PHONY : clean
clean :
	rm ./build/miniopera
