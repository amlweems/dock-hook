FROM golang AS build
WORKDIR /app
ADD go.mod .
ADD go.sum .
RUN go mod download
ADD . .
RUN go build

FROM golang
COPY --from=build /app/docker-hook /bin/
ENTRYPOINT ["/bin/docker-hook"]
