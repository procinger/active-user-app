package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"sync"
	"time"
)

type Visitor struct {
	LastSeen time.Time
}

type VisitorManager struct {
	mu       sync.Mutex
	visitors map[string]Visitor
}

func NewVisitorManager() *VisitorManager {
	return &VisitorManager{
		visitors: make(map[string]Visitor),
	}
}

func (vm *VisitorManager) Heartbeat(ip string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.visitors[ip] = Visitor{LastSeen: time.Now()}
}

func (vm *VisitorManager) Cleanup() {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	for ip, v := range vm.visitors {
		if time.Since(v.LastSeen) > 2*time.Minute {
			delete(vm.visitors, ip)
		}
	}
}

func (vm *VisitorManager) ActiveCount() int {
	vm.Cleanup()
	vm.mu.Lock()
	defer vm.mu.Unlock()
	return len(vm.visitors)
}

var pageTmpl = template.Must(template.New("page").Parse(`
<!DOCTYPE html>
<html lang="de">
<head>
  <meta charset="UTF-8">
  <title>Golang Example App</title>
</head>
<body>
  <h1>Golang Example App</h1>

  <h2>Active User</h2>
  <p id="users">Loading...</p>

  <h2>Environment Variables</h2>
  <pre>{{ .Env }}</pre>

  <script>
    setInterval(() => fetch("/ping"), 30000);
    fetch("/ping");

    async function updateUsers() {
      let res = await fetch("/active");
      let data = await res.json();
      document.getElementById("users").textContent =
        "Active User: " + data.active_users + " (has_active=" + data.has_active + ")";
    }
    setInterval(updateUsers, 10000);
    updateUsers();
  </script>
</body>
</html>
`))

func main() {
	manager := NewVisitorManager()

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		manager.Heartbeat(ip)
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/active", func(w http.ResponseWriter, r *http.Request) {
		count := manager.ActiveCount()
		response := map[string]interface{}{
			"active_users": count,
			"has_active":   count > 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		podIp := os.Getenv("POD_IP")
		if podIp == "" {
			podIp = "unknown"
		}

		podName := os.Getenv("POD_NAME")
		if podName == "" {
			podName = "unknown"
		}

		envVars := "POD_IP:\t\t" + podIp + "\nPOD_NAME:\t" + podName + "\n"

		data := struct {
			Env string
		}{
			Env: envVars,
		}
		pageTmpl.Execute(w, data)
	})

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	http.ListenAndServe(":"+appPort, nil)
}

func stringJoin(arr []string, sep string) string {
	result := ""
	for i, s := range arr {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
