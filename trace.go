package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", traceUI)
	router.HandleFunc("/{command}/{host}", doTrace)
	server := &http.Server{
		Addr:    ":3001",
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func traceUI(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(`<!DOCTYPE html><html><title>Traceroute</title>
	<style type="text/css">
	body {
		height: 100vw;
	}
	body, form {
		display: flex;
		flex-direction: column;
		align-items: stretch;
	}
	input, select {
		padding: 1em;
		line-height: 2em;
		border: none;
		border-radius: 5px;
	}
	iframe {
		border: none;
		flex: 1 100%;
	}
	</style>
	<form id="hostForm">
		<input id="host" type="text" value="1.1.1.1">
		<select>
			<option value="traceroute">traceroute</option>
			<option value="mtr">mtr</option>
			<option value="ping">ping</option>
		</select>
	</form>
	<iframe id="results"></iframe>
	<script type="text/javascript">
	document.bgColor = '#DDD';
	const switchTarget = () => {
		document.bgColor = '#DDD';
		document.querySelector('iframe').src = document.querySelector('select').value + '/' + document.querySelector('input').value;
	}
	document.querySelector('input').addEventListener('change', switchTarget );
	document.querySelector('form').addEventListener('submit', (e) => {
		switchTarget();
		e.preventDefault();
	} );
	document.querySelector('iframe').addEventListener('load', () => {
		document.querySelector('iframe').style.minHeight = document.querySelector('iframe').contentWindow.document.body.offsetHeight + "px";
		document.bgColor = '#DFD'
	})
	</script>
	</html>`))
}

func doTrace(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	host := vars["host"]
	command := vars["command"]

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "e", http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	defer conn.Close()
	bufrw.WriteString("HTTP/1.1 200 OK\r\n")
	bufrw.WriteString("Content-Type: text/plain; charset=us-ascii\r\n")
	bufrw.WriteString("X-Content-Type-Options: nosniff\r\n")
	bufrw.WriteString("Transfer-Encoding: identity\r\n")
	bufrw.WriteString("Connection: close\r\n\r\n")
	bufrw.Flush()

	cmd := &exec.Cmd{}
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()
	switch command {
	case "mtr":
		cmd = exec.CommandContext(ctx, "mtr", "-w", "-c 5", "-z", host)
	case "traceroute":
		cmd = exec.CommandContext(ctx, "traceroute", host)
	case "ping":
		cmd = exec.CommandContext(ctx, "ping", "-c", "30", "-A", "-w", "3", host)
	default:
		return
	}
	cmd.Stdout = bufrw
	cmd.Stderr = bufrw
	bufrw.WriteString(fmt.Sprintf("%q to %q...\r\n", command, host))
	bufrw.Flush()

	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}

	bufrw.Flush()
	cmd.Wait()
}
