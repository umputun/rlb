FROM umputun/baseimage:buildgo-latest as build

ARG COVERALLS_TOKEN
ARG CI
ARG TRAVIS
ARG TRAVIS_BRANCH
ARG TRAVIS_COMMIT
ARG TRAVIS_JOB_ID
ARG TRAVIS_JOB_NUMBER
ARG TRAVIS_OS_NAME
ARG TRAVIS_PULL_REQUEST
ARG TRAVIS_PULL_REQUEST_SHA
ARG TRAVIS_REPO_SLUG
ARG TRAVIS_TAG
ARG DRONE
ARG DRONE_TAG
ARG DRONE_COMMIT
ARG DRONE_BRANCH
ARG DRONE_PULL_REQUEST

WORKDIR /go/src/github.com/umputun/rlb
ADD . /go/src/github.com/umputun/rlb

# run tests
RUN cd app && go test ./...

# linters
RUN gometalinter --disable-all --deadline=300s --vendor --enable=vet --enable=vetshadow --enable=golint \
    --enable=staticcheck --enable=ineffassign --enable=errcheck --enable=unconvert \
    --enable=deadcode  --enable=gosimple --exclude=test --exclude=mock --exclude=vendor ./...

# coverage report
RUN mkdir -p target && /script/coverage.sh

# submit coverage to coverals if COVERALLS_TOKEN in env
RUN if [ -z "$COVERALLS_TOKEN" ] ; then \
    echo "coverall not enabled" ; \
    else goveralls -coverprofile=.cover/cover.out -service=travis-ci -repotoken $COVERALLS_TOKEN || echo "coverall failed!"; fi

RUN \
    version=$(git rev-parse --abbrev-ref HEAD)-$(git describe --abbrev=7 --always --tags)-$(date +%Y%m%d-%H:%M:%S) && \
    echo "version=$version" && \
    go build -o rlb -ldflags "-X main.revision=${version} -s -w" ./app


FROM umputun/baseimage:app

COPY --from=build /go/src/github.com/umputun/rlb/rlb /srv/rlb

EXPOSE 7070
WORKDIR /srv

CMD ["/srv/rlb"]
