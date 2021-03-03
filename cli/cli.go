package cli

import (
	"github.com/StarForger/neb-rcon/conn"
	"os"
	"log"
	"bufio" 																	// implements buffered I/O.
	"io"
	"fmt"
	"strings"
	"regexp"
)

const prompt = "[rcon] $ "

func Run(hostUri string, password string, in io.Reader, out io.Writer) {
	// Connect
	conn, err := conn.Dial(hostUri, password)
	if err != nil {
		log.Fatal("Failed to connect to RCON server: ", err)
	}
	defer conn.Close()

	// Input Scan
	input := bufio.NewScanner(in)
	out.Write([]byte(prompt))
	for input.Scan() {
		cmd := input.Text()
		response, err := conn.Execute(cmd)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Run error: ", err.Error())
			continue
		}

		print(out, response)
		out.Write([]byte(prompt))
	}

	if err := input.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error from input:", err)
	}
}

func Execute(hostUri string, password string, out io.Writer, command ... string) {
	// Connect	
	conn, err := conn.Dial(hostUri, password)
	if err != nil {
		log.Fatal("Failed to connect to RCON server: ", err)
	}
	defer conn.Close()

	// Send commands
	cmds := strings.Join(command, " ")
	response, err := conn.Execute(cmds)
	if err == io.EOF {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Execute error: ", err.Error())
		return
	}

	print(out, response)
}

func print(out io.Writer, msg string) {
	// strip out unknown character
	re := regexp.MustCompile("[§][\\w]")
	msg = re.ReplaceAllLiteralString(string(msg), "")

	fmt.Fprintln(out, msg)
}