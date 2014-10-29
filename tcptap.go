package main

import (
	"bufio"
	"fmt"
	"github.com/seanpont/gobro"
	"os"
	"strings"
	"time"
)

func testReader(args []string) {
	fmt.Print("\033[2J\033[1;1H:\n:\n$ ")

	go func() {
		for i := 0; i < 20; i++ {
			time.Sleep(time.Second)
			print(": hello %v", i)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			return
		}
		fmt.Print("\033[2J\033[1;1H:\n:" + strings.ToUpper(string(line)) + "\n$ ")
	}
}

func print(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	fmt.Print("\033[s\033[1;1H" + message + "\033[u")
}

func test2(args []string) {
	fmt.Print("\033[2J\033[1;1H")
	fmt.Print("one\ntwo\nthree\n")
	go func() {
		time.Sleep(time.Second)
		fmt.Print("\033[s" + strings.Repeat("\033[A\033[K", 4) + "\033[1;1H" + "four\nfive" + "\033[u")
	}()
	gobro.Prompt("$ ")
}

func main() {
	gobro.NewCommandMap(
		testReader,
		test2,
		simpleTapServer,
		connTapServer,
		connTapClient).Run(os.Args)
}
