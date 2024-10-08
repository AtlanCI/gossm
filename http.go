package gossm

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

func calculateServerUptime(statusAtTime []*statusAtTime) string {
	if len(statusAtTime) == 0 {
		return "unknown"
	}

	var sum float64

	for _, val := range statusAtTime {
		var i float64
		if val.Status {
			i = 1
		} else {
			i = 0
		}
		sum += i
	}

	return fmt.Sprintf("%.2f", sum/float64(len(statusAtTime))*100)
}

func lastStatus(statusAtTime []*statusAtTime) string {
	if len(statusAtTime) == 0 {
		return "Not yet checked"
	}
	lastChecked := statusAtTime[len(statusAtTime)-1]
	difference := time.Since(lastChecked.Time)
	status := "OK"
	if !lastChecked.Status {
		status = "ERR"
	}
	return fmt.Sprintf("%s, %.0f seconds ago", status, difference.Seconds())
}

func lastRTT(statusAtTime []*statusAtTime) string {
	if len(statusAtTime) == 0 {
		return "Not yet checked"
	}
	lastChecked := statusAtTime[len(statusAtTime)-1]
	return fmt.Sprintf("%s", lastChecked.TSS)
}

func RunHttp(address string, monitor *Monitor) {
	funcMap := template.FuncMap{
		"calculateServerUptime": calculateServerUptime,
		"lastStatus":            lastStatus,
		"lastRTT":               lastRTT,
	}

	t := template.Must(template.New("main").Funcs(funcMap).Parse(`<!DOCTYPE html>
<html lang="en">
  <head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
	<title>GOSSM - Dashboard</title>
	
    <!-- Bootstrap CSS -->
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0-beta/css/bootstrap.min.css" integrity="sha384-/Y6pD6FV/Vv2HJnA6t+vslU6fwYXjCFtcEpHbNJ0lyAFsXTsjBbfaDjzALeQsN6M" crossorigin="anonymous">
    <style> /* 当 uptime 低于 50% 时，按钮为红色 */

/* 低于 50% 显示红色按钮 */
a.btn[data-uptime][data-uptime^="100"] {
    background-color: green; /* 红色 */
    color: white;
}

/* 50% 及以上显示蓝色按钮 */
a.btn[data-uptime][data-uptime^="5"],
a.btn[data-uptime][data-uptime^="6"],
a.btn[data-uptime][data-uptime^="7"],
a.btn[data-uptime][data-uptime^="8"],
a.btn[data-uptime][data-uptime^="9"],
a.btn[data-uptime="100"] {
    background-color: #007bff; /* 蓝色 */
    color: white;
}
 </style>
  </head>
  <body>
	<div class="container">
		<br>
		<center><h1>Dashboard</h1></center>
		<hr>
		<div class="row">
			{{ range $server, $statusAtTime := .}}
			<div class="col-md-4">
				<div class="card" style="margin-top: 5px;">
					<div class="card-body">
						<h4 class="card-title">{{ $server.Name }}</h4>
						<p class="card-text">{{ $server }}<br>tested {{ len $statusAtTime }} times<br>{{ $statusAtTime | lastStatus }}</p>
                         <p class="card-text ">{{$statusAtTime | lastRTT }}</p>

						<a href="#" data-uptime="{{ $statusAtTime | calculateServerUptime }}" class="btn btn-danger">
                        {{ $statusAtTime | calculateServerUptime }}%</a>
					</div>
				</div>
			</div>
			{{ end }}
		</div>
	</div>
  </body>
</html>`))

	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		t.Execute(rw, monitor.serverStatusData.GetServerStatus())
	})

	http.HandleFunc("/json", func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")

		jsonBytes, err := json.Marshal(monitor.serverStatusData.GetServerStatus())
		if err != nil {
			jsonError, _ := json.Marshal(struct {
				Message string `json:"message"`
			}{Message: "Unable to format JSON."})

			rw.Write(jsonError)
			return
		}

		rw.Write(jsonBytes)
	})

	http.ListenAndServe(address, nil)
}
