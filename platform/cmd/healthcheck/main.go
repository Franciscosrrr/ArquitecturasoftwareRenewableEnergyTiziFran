package main

import (
	"net/http"
	"os"
	"time"
)

/**
 * Configura dependencias del proceso y ejecuta el servidor con apagado controlado.
 *
 * @method main
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func main() {
	req, _ := http.NewRequest("GET", "http://127.0.0.1:8080/health/ready", nil)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("HEALTHCHECK_TOKEN"))
	res, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		os.Exit(1)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		os.Exit(1)
	}
}
