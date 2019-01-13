EXE?=

ifeq ($(OS),Windows_NT)
	ifeq ($(EXE),)
    	EXE=".exe"
	endif
endif

all: gitrepo$(EXE)

gitrepo$(EXE): main.go
	go build -ldflags "-w -extldflags -static" \
	-tags netgo -installsuffix netgo -o $@ main.go

cross:
	GOOS=linux GOARCH=amd64 EXE=.linux $(MAKE) --no-print-directory
	GOOS=windows GOARCH=amd64 EXE=.exe $(MAKE) --no-print-directory
	GOOS=darwin GOARCH=amd64 EXE=.mac $(MAKE) --no-print-directory

clean:
	rm -f gitrepo gitrepo.exe gitrepo.linux gitrepo.mac
