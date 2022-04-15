ARG GO_VERSION=1.18.0-v2
ARG ALPINE_VERSION=3.14

FROM registry.lucky-team.pro/luckyads/go.docker-images/alpine:${GO_VERSION} AS build
WORKDIR /opt/go.domain-checker
COPY . .
ARG ACTION
RUN make $ACTION

FROM alpine:${ALPINE_VERSION}
WORKDIR /opt/go.domain-checker
COPY --from=build /opt/go.domain-checker/bin/go.domain-checker .
CMD ["./go.domain-checker"]
