# Ultron Observer

Ultron Observer is a tool designed to monitor in-cluster resources for Ultron. It integrates with various cluster APIs to provide real-time data on cluster resources to Ultron. Ultron Observer is built with Go and can be run as a standalone application or within a Docker container.

## Prerequisites

- Go 1.23 or higher
- Docker (if you want to run the application in a container)

## Environment Variables

The application requires the following environment variables to be set:

- `EMMA_CLIENT_ID`: Your Emma API client ID
- `EMMA_CLIENT_SECRET`: Your Emma API client secret

## Installation

### Clone the repository

```sh
git clone https://github.com/be-heroes/ultron-observer
cd ultron-observer
```

### Set up environment variables

```sh
export EMMA_CLIENT_ID=your_client_id
export EMMA_CLIENT_SECRET=your_client_secret
```

### Build the application

```sh
go build -o main main.go
```

### Run the application

```sh
./main
```

## Docker

To build and run the application using Docker.

### Build the Docker image

```sh
docker build -t ultron-observer:latest .
```

### Run the Docker container

```sh
docker run -e EMMA_CLIENT_ID=your_client_id -e EMMA_CLIENT_SECRET=your_client_secret ultron-observer:latest
```

## Additional links

- [Project Ultron => Abstract](https://github.com/be-heroes/ultron/blob/main/docs/ultron_abstract.md)
- [Project Ultron => Algorithm](https://github.com/be-heroes/ultron/blob/main/docs/ultron_algorithm.md)
- [Project Ultron => WebHookServer Sequence Diagram](https://github.com/be-heroes/ultron/blob/main/docs/ultron.png)
