package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/websocket"
)

const homeText = `
<html><head>
<script type="text/javascript" src="http://ajax.googleapis.com/ajax/libs/jquery/1.4.2/jquery.min.js"></script>
<script type="text/javascript">
    $(function() {
        var conn = new WebSocket("ws://{{$}}/ws");
        conn.onclose = function(evt) {
            $("<div><b>closed</b></div>").appendTo($("#log"));
        }
        conn.onmessage = function(evt) {
            $("<div/>").text(evt.data).appendTo($("#log"));
        }
        $("#send").submit(function() {
            conn.send(JSON.stringify({
                    name: "send", 
                    command: {
                        room: $("#send_room").val(),
                        message: $("#send_message").val(),
                    }}));
            return false
        });
        $("#join").submit(function() {
                conn.send(JSON.stringify({
                    name: "join", 
                    command: {
                        room: $("#join_room").val(),
                    }}));
            return false
        });
    });
</script>
</head>
<body>
<form id="join">
    <input type="text" id="join_room" value="go-nuts"/>
    <input type="submit" value="Join" />
</form>
<form id="send">
    <input type="text" id="send_room" value="go-nuts"/>
    <input type="text" id="send_message" value="Hello World!"/>
    <input type="submit" value="Send" />
</form>
<div id="log"></div>
</body>
`

var (
	addr      = flag.String("addr", ":8080", "http service address")
	homeTempl = template.Must(template.New("").Parse(homeText))
)

func homeHandler(c http.ResponseWriter, req *http.Request) {
	homeTempl.Execute(c, req.Host)
}

type joinCommand struct {
	Room string
}

type sendCommand struct {
	Room    string
	Message string
}

var commands = map[string]func() interface{}{
	"join": func() interface{} { return new(joinCommand) },
	"send": func() interface{} { return new(sendCommand) },
}

func readCommand(ws *websocket.Conn) interface{} {
	var command struct {
		Name    string
		Command json.RawMessage
	}
	if err := ws.ReadJSON(&command); err != nil {
		return err
	}
	newFn := commands[command.Name]
	if newFn == nil {
		return errors.New("unknown command " + command.Name)
	}
	v := newFn()
	if err := json.Unmarshal([]byte(command.Command), v); err != nil {
		return err
	}
	return v
}

var upgrader = &websocket.Upgrader{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	for {
		switch c := readCommand(ws).(type) {
		case *joinCommand:
			ws.WriteMessage(websocket.TextMessage, []byte("join room "+c.Room))
		case *sendCommand:
			ws.WriteMessage(websocket.TextMessage, []byte("send "+c.Message+" to room "+c.Room))
		case error:
			log.Println(c)
			return
		}
	}
}

func wsHandler1(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	for {
		var command struct {
			Name    string
			Command json.RawMessage
		}
		if err := ws.ReadJSON(&command); err != nil {
			log.Println(err)
			return
		}
		switch command.Name {
		case "join":
			var c joinCommand
			if err := json.Unmarshal([]byte(command.Command), &c); err != nil {
				log.Println(err)
				return
			}
			ws.WriteMessage(websocket.TextMessage, []byte("join room "+c.Room))
		case "send":
			var c sendCommand
			if err := json.Unmarshal([]byte(command.Command), &c); err != nil {
				log.Println(err)
				return
			}
			ws.WriteMessage(websocket.TextMessage, []byte("send "+c.Message+" to room "+c.Room))
		default:
			ws.WriteMessage(websocket.TextMessage, []byte("unknown"))
		}
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/ws", wsHandler)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
