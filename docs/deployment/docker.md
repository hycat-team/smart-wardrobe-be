# Docker Guide

The system is packaged via a Dockerfile using a Multi-stage build model to optimize the output image size.

## 1. Local Build
Run the following command to build the production image on your local machine:
```bash
make docker-build
```

## 2. Docker Compose
The system uses different docker-compose files depending on the environment:
*   `docker-compose.yml`: Development environment (includes local Postgres, Redis, RabbitMQ).
*   `docker-compose.prod.yml`: Production environment.
