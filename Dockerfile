# dynamic config
ARG             BUILD_DATE
ARG             VCS_REF
ARG             VERSION

# build
FROM            golang:1.22-alpine as builder
RUN             apk add --no-cache git gcc musl-dev make
ENV             GO111MODULE=on
WORKDIR         /go/src/ultre.me/speechotron
COPY            go.* ./
RUN             go mod download
COPY            . ./
RUN             make install

# minimalist runtime
FROM            alpine:3.19
LABEL           org.label-schema.build-date=$BUILD_DATE \
                org.label-schema.name="speechotron" \
                org.label-schema.description="" \
                org.label-schema.url="https://ultre.me/speechotron/" \
                org.label-schema.vcs-ref=$VCS_REF \
                org.label-schema.vcs-url="https://github.com/ultreme/speechotron" \
                org.label-schema.vendor="Manfred Touron" \
                org.label-schema.version=$VERSION \
                org.label-schema.schema-version="1.0" \
                org.label-schema.cmd="docker run -i -t --rm ultreme/speechotron" \
                org.label-schema.help="docker exec -it $CONTAINER speechotron --help"
RUN             apk add --no-cache sox
COPY            --from=builder /go/bin/speechotron /bin/
WORKDIR         /speechotron
COPY            ./pronounce ./pronounce
COPY            ./say ./say
ENTRYPOINT      ["/bin/speechotron"]
#CMD             []
EXPOSE          8000