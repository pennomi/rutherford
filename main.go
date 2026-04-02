package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"net/http"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

//go:embed all:ui/build
var uiFiles embed.FS

func spaFileServer(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/" && len(path) > 1 {
			_, err := fs.Stat(root, path[1:])
			if err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "path to kubeconfig file (uses in-cluster config if not set)")
	noAuth := flag.Bool("no-auth", false, "disable authentication (requires --kubeconfig)")
	flag.Parse()

	if *noAuth && *kubeconfig == "" {
		panic("--no-auth requires --kubeconfig (refusing to disable auth in-cluster)")
	}

	ctx := context.Background()

	auth := NewAuthenticator(ctx, *noAuth)
	defer auth.Close()

	var config *rest.Config
	var err error
	if *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic("failed to build config from kubeconfig: " + err.Error())
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic("failed to get in-cluster config: " + err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic("failed to create kubernetes client: " + err.Error())
	}

	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		panic("failed to create metrics client: " + err.Error())
	}

	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		panic("failed to create metadata client: " + err.Error())
	}

	hub := NewHub()
	watcher := NewWatcher(clientset, metricsClient, metadataClient, hub)
	watcher.Start(ctx)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/auth/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(auth.AuthConfigJSON())
	})

	mux.HandleFunc("GET /api/auth/check", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		err := auth.ValidateToken(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /ws", HandleWebSocket(auth, hub, watcher))
	mux.HandleFunc("GET /ws/logs", HandleLogStream(auth, clientset))
	uiRoot, err := fs.Sub(uiFiles, "ui/build")
	if err != nil {
		panic("failed to access embedded UI files: " + err.Error())
	}
	spaHandler := spaFileServer(uiRoot)

	panic(http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/auth/config" || path == "/api/auth/check" ||
			path == "/ws" || path == "/ws/logs" {
			mux.ServeHTTP(w, r)
			return
		}
		spaHandler.ServeHTTP(w, r)
	})))
}
