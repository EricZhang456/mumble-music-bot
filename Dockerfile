FROM golang:1.25.3-alpine AS builder
WORKDIR /src
RUN --mount=type=cache,target=/var/cache/apk apk add opus opus-dev build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /src/out/mumble-music-bot .

FROM alpine:3
ARG TARGETARCH

ENV COMMAND_PREFIX="!"
ENV BOT_DB_PATH="/db/db.sqlite3"
ENV MUMBLE_PORT=64738
ENV MUSIC_PATH="/data"

RUN mkdir /app /db /data
COPY --from=builder /src/out/mumble-music-bot /app

RUN --mount=type=cache,target=/var/cache/apk <<EOF
apk add ffmpeg
if [ "$TARGETARCH" != "amd64" ] && [ "$TARGETARCH" != "386" ]; then
    apk add opus
fi
EOF

VOLUME [ "/db", "/data" ]

CMD [ "/app/mumble-music-bot" ]
