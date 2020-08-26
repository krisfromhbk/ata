# Accept the Go version for the image to be set as a build argument.
ARG GO_VERSION=1.14.7

# First stage: build the executable.
FROM golang:${GO_VERSION}-alpine AS builder

LABEL maintainer="Kirill Samoilov <kreisnews@gmail.com>"

ENV GO111MODULE=on

# Create the user and group files that will be used in the running container to
# run the process as an unprivileged user.
RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

# Git is required for fetching the dependencies.
RUN apk add --no-cache git

# Set the working directory
WORKDIR /src

# Fetch dependencies first; they are less susceptible to change on every build
# and will therefore be cached for speeding up the next build
COPY go.mod go.sum ./
RUN go mod download

# Import the code from the context.
COPY . .

# Build the executable to `/app`. Mark the build as statically linked.
RUN CGO_ENABLED=0 go build \
    -installsuffix 'static' \
    -o /app cmd/server/main.go

# Final stage: the running container.
FROM scratch AS final

# Import the user and group files from the first stage.
COPY --from=builder /user/group /user/passwd /etc/

# Import the compiled executable from the first stage.
COPY --from=builder /app /app

# Declare the port on which the webserver will be exposed.
# As we're going to run the executable as an unprivileged user, we can't bind
# to ports below 1024.
EXPOSE 9000

# Perform any further action as an unprivileged user.
USER nobody:nobody

# Run the compiled binary.
ENTRYPOINT ["/app"]