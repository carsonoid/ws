package main

import (
	"time"
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"github.com/chzyer/readline"
	"github.com/rgamba/evtwebsocket"
	"github.com/spf13/cobra"
	"golang.org/x/net/websocket"
)

const Version = "0.2.1"

var options struct {
	origin       string
	printVersion bool
	stdin        bool
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "ws URL",
		Short: "websocket tool",
		Run:   root,
	}
	rootCmd.Flags().StringVarP(&options.origin, "origin", "o", "", "websocket origin")
	rootCmd.Flags().BoolVarP(&options.stdin, "stdin", "i", false, "read input from stdin not interactive")
	rootCmd.Flags().BoolVarP(&options.printVersion, "version", "v", false, "print version")

	rootCmd.Execute()
}

func root(cmd *cobra.Command, args []string) {
	if options.printVersion {
		fmt.Printf("ws v%s\n", Version)
		os.Exit(0)
	}

	if len(args) != 1 {
		cmd.Help()
		os.Exit(1)
	}

	dest, err := url.Parse(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var origin string
	if options.origin != "" {
		origin = options.origin
	} else {
		originURL := *dest
		if dest.Scheme == "wss" {
			originURL.Scheme = "https"
		} else {
			originURL.Scheme = "http"
		}
		origin = originURL.String()
	}

	var historyFile string
	user, err := user.Current()
	if err == nil {
		historyFile = filepath.Join(user.HomeDir, ".ws_history")
	}

	if options.stdin {
		connected := false
		fmt.Println("Reading from stdin:")
		c := evtwebsocket.Conn{
			OnConnected: func(w *websocket.Conn) {
				connected = true
				fmt.Println("Connected")
			},
			OnMessage: func(msg []byte) {
				fmt.Printf("Received message: %s\n", msg)
			},
			OnError: func(err error) {
				fmt.Printf("** ERROR **\n%s\n", err.Error())
				os.Exit(1)
			},
		}
		// Connect
		c.Dial(dest.String())

		// Wait for connection
		for {
			if connected {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}

		// read from stdin and send as lines show up
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			in := scanner.Text()
			fmt.Printf("Got stdin: %s\n", in)
			msg := evtwebsocket.Msg{
				Body: []byte(in),
				Callback: func(resp []byte) {
					// This function executes when the server responds
					fmt.Printf("Got response: %s\n", resp)
				},
			}
			c.Send(msg)
		}
	} else {
		err = connect(dest.String(), origin, &readline.Config{
			Prompt:      "> ",
			HistoryFile: historyFile,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if err != io.EOF && err != readline.ErrInterrupt {
				os.Exit(1)
			}
		}
	}
}
