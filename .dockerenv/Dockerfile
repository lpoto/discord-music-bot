FROM golang:alpine as builder

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
# gcc is required for cgo
RUN apk update && apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev \
    && update-ca-certificates

# Create our working directory
RUN mkdir /app
# Create appuser
RUN adduser -D -g '' appuser

ADD ./src /app
WORKDIR /app

ADD ./assets /go/assets

# Build the binary
RUN GOOS=linux go build -ldflags="-w -s" -a -o /go/bin/music_bot .


FROM opencoconut/ffmpeg

# Import from builder.
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

# Copy our static executable
COPY --from=builder /go/bin/music_bot /go/bin/music_bot
COPY --from=builder /go/assets /go/assets

WORKDIR /go/bin

USER appuser

ENTRYPOINT ["/go/bin/music_bot"]

