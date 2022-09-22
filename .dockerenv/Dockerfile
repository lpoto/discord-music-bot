FROM golang:alpine as builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Create our working directory
RUN mkdir /app
# Create appuser
RUN adduser -D -g '' appuser

ADD . /app
WORKDIR /app/src

# Build the binary, add CGO_FLAGS to avoid gcc errors
RUN CGO_CFLAGS="-w -Os" CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o /go/bin/music_bot .
RUN mkdir /go/bin/conf


FROM jrottenberg/ffmpeg

# Import from builder.
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

# Copy our static executable
COPY --from=builder /go/bin/loans /go/bin/music_bot
COPY --from=builder /go/bin/conf /go/bin/conf

WORKDIR /go/bin

USER appuser

ENTRYPOINT ["/go/bin/music_bot"]

