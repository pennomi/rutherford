FROM node:22-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ .
RUN npm run build

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY --from=ui-builder /app/ui/build ./ui/build/
RUN CGO_ENABLED=0 GOOS=linux go build -o rutherford .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/rutherford /rutherford
ENTRYPOINT ["/rutherford"]
