FROM golang:1.3.3-onbuild

MAINTAINER Sean Pont <seanpont@gmail.com>

EXPOSE 8080
ENTRYPOINT app connTapServer 8080
