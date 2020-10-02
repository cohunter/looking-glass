package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/{command}/{host}", doTrace)
	router.Methods("GET").HandlerFunc(traceUI)
	router.Methods("POST").HandlerFunc(doTrace)
	server := &http.Server{
		Addr:    ":3001",
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func traceUI(w http.ResponseWriter, req *http.Request) {
	remote_host := "8.8.8.8"
	if len(req.Header.Values("Cf-Connecting-Ip")) == 1 {
		remote_host = req.Header.Values("Cf-Connecting-Ip")[0]
	}
	const tpl = `<!DOCTYPE html><html>
	<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1">
	<title>Traceroute</title>
	<style type="text/css">
	body {
		height: 95vh;
		flex-direction: column;
	}
	body, form {
		display: flex;
		align-items: stretch;
	}
    form {
        flex-direction: row;
        flex-wrap: wrap;
    }
    form {
        position: fixed;
        bottom: 0.1em;
        left: 0.1em;
        right: 0.1em;
    }
    input {
		flex: 1 100%;
    }
	input, select, button {
		padding: 1em;
		line-height: 2em;
		border: none;
		border-radius: 5px;
	}
    select, button {
		flex: 1 0;
		min-width: 150px;
		background: darkseagreen;
    }
	iframe {
		border: none;
		flex: 1 100%;
	}
	</style>
	<form id="hostForm" method="POST">
		<input name="host" type="text" value="{{ .IP }}">
		<select name="command">
			<option value="traceroute">traceroute</option>
			<option value="mtr">mtr</option>
			<option value="ping">ping</option>
		</select>
		<button type="submit">Start</button>
	</form>
	<iframe id="results" src="traceroute/{{ .IP }}"></iframe>
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
		document.querySelector('iframe').style.minHeight = (100 + document.querySelector('iframe').contentWindow.document.body.offsetHeight) + "px";
		document.bgColor = '#DFD';
	})
	</script>
	</html>`
	t, err := template.New("webpage").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}
	data := struct {
		IP string
	}{
		IP: remote_host,
	}
	err = t.Execute(w, data)
}

func doTrace(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	host, ok := vars["host"]
	command := vars["command"]
	if !ok {
		req.ParseForm()
		if len(req.Form["host"]) != 1 || len(req.Form["command"]) != 1 {
			http.Error(w, "bad request", 400)
			return
		}
		host = req.Form["host"][0]
		command = req.Form["command"][0]
	}

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
	cmd.Stdout, cmd.Stderr = bufrw, bufrw
	bufrw.WriteString(fmt.Sprintf("%q to %q...\r\n", command, host))
	bufrw.Flush()

	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}

	bufrw.Flush()
	cmd.Wait()
}
