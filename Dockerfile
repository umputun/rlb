FROM umputun/baseimage:buildgo-latest as build

ARG GIT_BRANCH
ARG GITHUB_SHA
ARG CI

ENV GOFLAGS="-mod=vendor"
ENV CGO_ENABLED=0

ADD . /build
WORKDIR /build

RUN \
    if [ -z "$CI" ] ; then \
    echo "runs outside of CI" && version=$(git rev-parse --abbrev-ref HEAD)-$(git log -1 --format=%h)-$(date +%Y%m%dT%H:%M:%S); \
    else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version" && \
    cd app && go build -o /build/rlb -ldflags "-X main.revision=${version} -s -w"


FROM umputun/baseimage:app-latest
COPY --from=build /build/rlb /srv/rlb
EXPOSE 7070
WORKDIR /srv
CMD ["/srv/rlb"]