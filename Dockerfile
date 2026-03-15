FROM golang:1.25-alpine AS builder

WORKDIR /app
RUN apk add --no-cache ca-certificates upx
RUN echo "appgroup:x:1000:" > /etc/group_app && \
    echo "appuser:x:1000:1000:App User:/:/sbin/nologin" > /etc/passwd_app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main ./cmd/server/main.go
FROM scratch

WORKDIR /app
COPY --from=builder /etc/passwd_app /etc/passwd
COPY --from=builder /etc/group_app /etc/group
COPY --from=builder --chown=appuser:appgroup /app/main .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER appuser
EXPOSE 8080

CMD [ "./main" ]