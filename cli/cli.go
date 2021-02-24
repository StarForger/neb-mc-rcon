package cli

import (
	"github.com/StarForger/neb-rcon/conn"
	"os"
	"log"
	"bufio" 																	// implements buffered I/O.
	"io"
	"fmt"
	"strings"
	"net"
)

const prompt = "$> "

func Run(string host, string password, in io.Reader, out io.Writer) {
	// Connect
	conn, err := connection.Dial(host, password)
	if err != nil {
		log.Fatal("Failed to connect to RCON server", err)
	}
	defer conn.Close()

	// Input Scan
	input := bufio.NewScanner(in)
	out.Write([]byte(prompt))
	// TODO EOF?
	for input.Scan() {
		cmd := scanner.Text()
		response, err := connection.Execute(cmd); err != nil {
			fmt.Fprintln(os.Stderr, "Run error: ", err.Error())
			continue
		}

		fmt.Fprintln(out, response)
		out.Write([]byte(prompt))
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error from input:", err)
	}
}

func Execute(string host, string password, out io.Writer, string command ...) {
	// Connect	
	conn, err := connection.Dial(host, password)
	if err != nil {
		log.Fatal("Failed to connect to RCON server", err)
	}
	defer conn.Close()

	// Send commands
	cmds := strings.Join(command, " ")
	response, err := connection.Execute(cmds); err != nil {
		fmt.Fprintln(os.Stderr, "Execute error: ", err.Error())
		return
	}

	fmt.Fprintln(out, resp)
}