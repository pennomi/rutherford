package main

import (
	"bufio"
	"context"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"nhooyr.io/websocket"
)

func HandleLogStream(auth Authenticator, clientset *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace := r.URL.Query().Get("namespace")
		pod := r.URL.Query().Get("pod")
		container := r.URL.Query().Get("container")

		if namespace == "" || pod == "" || container == "" {
			http.Error(w, "namespace, pod, and container query params required", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			http.Error(w, "websocket accept failed", http.StatusBadRequest)
			return
		}

		_, tokenBytes, err := conn.Read(r.Context())
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "failed to read auth token")
			return
		}
		err = auth.ValidateToken(string(tokenBytes))
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "invalid auth token")
			return
		}

		ctx := conn.CloseRead(r.Context())

		tailLines := int64(200)
		stream, err := clientset.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
			Container:  container,
			Follow:     true,
			TailLines:  &tailLines,
			Timestamps: true,
		}).Stream(ctx)
		if err != nil {
			conn.Close(websocket.StatusInternalError, "failed to open log stream")
			return
		}
		defer stream.Close()

		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, scanner.Bytes())
			cancel()
			if err != nil {
				return
			}
		}

		conn.Close(websocket.StatusNormalClosure, "")
	}
}
